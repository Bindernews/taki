// An Imager runs kubectl and communicates with the server to image a given container.
// Multiple imagers may be run at once.
package imager

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bindernews/taki/pkg/rpcfs"
	"github.com/bindernews/taki/pkg/task"
	"github.com/bindernews/taki/pkg/tkserver"
)

const CONTAINER_NAME_PREFIX = "Defaulting debug container name to "
const taskLocalMeta = "building local metadata"
const taskGenerateDiff = "generating diff from base image"
const taskTarFiles = "collecting changed files"
const taskDownload = "downloading archive"

type ImagerConfig struct {
	// Command that runs 'kubectl'
	KubectlCmd []string
	// Pod name to image
	Pod string
	// Container on the pod
	Container string
	// Debug image name and version (default: "taki-collector")
	DebugImage string
	// List of paths to ignore
	Ignored []string
	// Image cache instance
	MetaCache *ImageCache
	// Base image path
	BaseImage string
}

// Returns a copy of the config with default values set if they weren't already.
func (c ImagerConfig) Defaults() ImagerConfig {
	if c.DebugImage == "" {
		c.DebugImage = "taki-collector"
	}
	if c.MetaCache == nil {
		c.MetaCache = &ImageCache{}
	}
	return c
}

type Imager struct {
	config ImagerConfig
	// Debug container name, set post-Start
	debugContainerName string
	// Subprocess context
	ctx context.Context
	// Cancel func
	cancelFn context.CancelFunc
	// rpc client
	client *ClientApi
	// RpcFS client
	rfs *rpcfs.RpcFs
	// Current task name
	currentTask string
	// Current progress
	curProgress *task.AtomicFloat64
	// Update channel
	updateC chan struct{}
}

// Returns a new imager created with the given configuration.
func NewImager(parent context.Context, config ImagerConfig) *Imager {
	m := &Imager{
		config:      config.Defaults(),
		currentTask: "",
		curProgress: task.NewF64(0),
		updateC:     make(chan struct{}),
	}
	// Initialize some internal variables
	m.ctx, m.cancelFn = context.WithCancel(parent)
	return m
}

func (m *Imager) Start() (err error) {
	const OUTPUT_PATH = "/root/root.tar.xz"
	var pio *ProcIO
	var possibleRoots []string
	var progress float64

	// Build list of all arguments
	allArgs := append(
		m.config.KubectlCmd,
		"debug",
		m.config.Pod,
		"-it",
		"--container="+m.config.Container,
		"--image="+m.config.DebugImage,
	)

	// Get pod metadata and verify that container exists, etc. Get image name.
	// TODO
	// Get base image and build DirInfo for it. Use cache in case of batch processing.
	metaReq := m.config.MetaCache.Request(m.config.BaseImage)
	// TODO
	// Start kubectl debug -it and setup stdio pipes
	proc := exec.CommandContext(m.ctx, allArgs[0], allArgs[1:]...)
	// Parse stdout for debug container name and for server start
	if pio, err = NewProcIO(proc); err != nil {
		return
	}
	m.debugContainerName, err = WaitForServerStart(pio.Reader)
	if err != nil {
		return
	}

	// Server is running on remote, setup ClientApi
	m.client = NewClientApi(m.ctx, pio)
	m.rfs = rpcfs.NewRpcFs(m.ctx, m.client.Client)

	// Get PIDs and determine/guess which ones are from target container??? Maybe prompt user?
	// TODO allow user to filter/select based on process command line
	if possibleRoots, err = m.client.GetTargetRoots(); err != nil {
		return
	}
	if len(possibleRoots) != 1 {
		err = fmt.Errorf("multiple roots found, please specify process name")
		return
	}

	// TODO get mounts so we can exlude them
	mounts := make([]string, 0)

	// Wait for DirMeta to be ready
	m.currentTask = taskLocalMeta
	m.setProgress(-1)
	select {
	case <-metaReq.Done():
		err = metaReq.Err()
	case <-m.ctx.Done():
		err = m.ctx.Err()
	}
	if err != nil {
		return
	}

	// Set config
	excludes := append(mounts, m.config.Ignored...)
	conf := tkserver.ServerConfig{
		Output:  OUTPUT_PATH,
		Root:    possibleRoots[0],
		Exclude: excludes,
	}
	if err = m.client.SetConfig(&conf); err != nil {
		return
	}
	// Have server diff and produce tar
	m.currentTask = taskGenerateDiff
	m.setProgress(-1)
	if err = m.client.GenerateDiff(metaReq.Value()); err != nil {
		return
	}

	m.currentTask = taskTarFiles
	m.setProgress(-1)
	if err = m.client.TarStart(); err != nil {
		return
	}
	// Wait for task to complete
	for {
		if progress, err = m.client.TarProgress(); err != nil {
			return
		}
		m.setProgress(progress)
		if progress >= 1.0 {
			break
		}
	}

	// Download tar and name it <pod_name>_<image_name>.tar.xz
	m.currentTask = taskDownload
	m.setProgress(0)
	dstName := m.GetOutputName()
	if err = m.DownloadFile(OUTPUT_PATH, dstName); err != nil {
		return
	}

	// Profit!
	return
}

// Returns the tar file that will be created
func (m *Imager) GetOutputName() string {
	return fmt.Sprintf("%s_%s.tar.xz", m.config.Pod, m.config.Container)
}

// Close the update channel so the imager does not block.
func (m *Imager) CloseUpdates() {
	close(m.updateC)
}

// Gets the update channel, which will trigger when a progress update
// occurs. Current progress and current task can be obtained with
// the relevant methods.
func (m *Imager) Updates() <-chan struct{} {
	return m.updateC
}

// Returns the current task name
func (m *Imager) GetTask() string {
	return m.currentTask
}

// Returns the progress amount on the current task, if applicable, or -1
// if the current task has no progress indicator.
func (m *Imager) GetProgress() float64 {
	return m.curProgress.Get()
}

// Helper to internally set the progress
func (m *Imager) setProgress(p float64) {
	m.curProgress.Set(p)
	// Send to update channel
	m.updateC <- struct{}{}
}

// Download a file
func (m *Imager) DownloadFile(remotePath, localPath string) error {
	// Open source
	src, err := m.rfs.OpenRead(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()
	// Open destination
	dst, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	// Copy task, later we can make progress accessible
	t := tkserver.NewTaskCopy(src, dst)
	go t.Start()
	for {
		prog := t.GetProgress()
		m.setProgress(prog)
		if prog >= 1 {
			break
		}
	}

	if t.Err() != nil {
		return t.Err()
	}
	return nil
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
		if ln == tkserver.SERVER_START_LINE {
			return
		}
	}
}
