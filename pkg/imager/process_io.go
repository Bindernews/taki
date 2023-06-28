package imager

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
)

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
