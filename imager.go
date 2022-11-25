package taki

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

const CONTAINER_NAME_PREFIX = "Defaulting debug container name to "

type ImagerProgress struct {
	// Name of the task the imager is working on
	Task string
	// Progress amount
	Amount float64
}

type Imager struct {
	// Command that runs 'kubectl'
	KubectlCmd []string
	// Pod name to image
	Pod string
	// Container on the pod
	Container string
	// Image cache instance
	MetaCache *ImageCache
	// Base image path
	BaseImage string
	// Progress channel
	progressC chan ImagerProgress
	// Debug container name, set post-Start
	debugContainerName string
	// Subprocess context
	ctx context.Context
	// Cancel func
	cancelFn context.CancelFunc
	// rpc client
	client *ClientApi
}

func (m *Imager) Start(parent context.Context) (err error) {
	var pio *ProcIO
	var possibleRoots []string
	var collectTask TaskHandle
	var progress float64

	// Initialize some internal variables
	m.ctx, m.cancelFn = context.WithCancel(parent)
	m.progressC = make(chan ImagerProgress)
	// Build list of all arguments
	allArgs := append(m.KubectlCmd, "") // TODO

	// Get pod metadata and verify that container exists, etc. Get image name.
	// TODO
	// Get base image and build DirInfo for it. Use cache in case of batch processing.
	metaReq := m.MetaCache.Request(m.BaseImage)
	// TODO
	// Start kubectl debug -it and setup stdio pipes
	proc := exec.CommandContext(m.ctx, allArgs[0], allArgs[1:]...)
	// Parse stdout for debug container name and for server start
	if pio, err = NewProcIO(proc); err != nil {
		return
	}
	m.debugContainerName, err = WaitForServerStart(pio.Reader)

	// Server is running on remote, setup ClientApi
	m.client = NewClientApi(m.ctx, pio)

	// Get PIDs and determine/guess which ones are from target container??? Maybe prompt user?
	// TODO allow user to filter/select based on process command line
	if possibleRoots, err = m.client.GetTargetRoots(); err != nil {
		return
	}
	if len(possibleRoots) != 1 {
		err = fmt.Errorf("multiple roots found, please specify process name")
		return
	}

	// Wait for DirMeta to be ready
	select {
	case <-metaReq.Done():
		err = metaReq.Err()
	case <-m.ctx.Done():
		err = m.ctx.Err()
	}
	if err != nil {
		return
	}

	// Have server diff and produce tar
	if err = m.client.GenerateDiff(metaReq.Value()); err != nil {
		return
	}
	if err = m.client.CollectFilesStart(&collectTask); err != nil {
		return
	}
	// Wait for task to complete
	for {
		if progress, err = m.client.TaskProgress(collectTask); err != nil {
			return
		}
		// Share progress update
		m.progressC <- ImagerProgress{
			Task:   "tar",
			Amount: progress,
		}
		if progress >= 1.0 {
			break
		}
	}

	// Download tar and name it <pod_name>_<image_name>.tar.xz

	// Profit!
	return
}

// Reads lines from the reader until it has read the auto-generated debug container name
// and the server start message. Returns the container name.
func WaitForServerStart(rd *bufio.Reader) (containerName string, err error) {
	for {
		var ln string
		if ln, err = rd.ReadString('\n'); err != nil {
			return
		}
		if strings.HasPrefix(ln, CONTAINER_NAME_PREFIX) {
			s2 := strings.TrimPrefix(ln, CONTAINER_NAME_PREFIX)
			s2 = s2[:len(s2)-1]
			containerName = s2
		}
		if ln == SERVER_START_LINE {
			return
		}
	}
}

// Takes an exec.Cmd and wraps the Stdin and Stdout/Stderr in buffered
// readers/writers. Stdout and Stderr are combined into one stream.
type ProcIO struct {
	*bufio.ReadWriter
	// Process
	proc *exec.Cmd
	// Things to close
	toClose []io.Closer
}

func NewProcIO(proc *exec.Cmd) (*ProcIO, error) {
	inRaw, err := proc.StdinPipe()
	if err != nil {
		return nil, err
	}
	inBuf := bufio.NewWriter(inRaw)

	outRd, outWr := io.Pipe()
	proc.Stderr = outWr
	proc.Stdout = outWr
	parent := bufio.NewReadWriter(bufio.NewReader(outRd), inBuf)
	return &ProcIO{
		ReadWriter: parent,
		proc:       proc,
		toClose:    []io.Closer{inRaw, outRd, outWr},
	}, nil
}

func (p *ProcIO) Close() error {
	errList := make([]error, 0)
	for _, c := range p.toClose {
		if err := c.Close(); err != nil {
			errList = append(errList, err)
		}
	}
	if len(errList) > 0 {
		return fmt.Errorf("%+v", errList)
	} else {
		return nil
	}
}
