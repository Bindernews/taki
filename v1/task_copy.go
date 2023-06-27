package taki

import (
	"io"
	"sync/atomic"

	"github.com/bindernews/taki/task"
)

type TaskCopy struct {
	*task.BaseTask
	src io.ReadSeekCloser
	dst io.WriteCloser
	// Progress bytes
	bytesCopied int64
	// Total size of the src file to copy
	totalSize int64
}

func NewTaskCopy(src io.ReadSeekCloser, dst io.WriteCloser) *TaskCopy {
	return &TaskCopy{
		BaseTask:    task.NewBaseTask(),
		src:         src,
		dst:         dst,
		bytesCopied: 0,
		totalSize:   0,
	}
}

func (t *TaskCopy) Start() task.Void {
	var err error

	if t.totalSize, err = t.src.Seek(0, io.SeekEnd); err != nil {
		return t.Fail(err)
	}
	if _, err = t.src.Seek(0, io.SeekStart); err != nil {
		return t.Fail(err)
	}

	var buf [4096]byte
	var ns, nd int
	for {
		if ns, err = t.src.Read(buf[:]); err != nil {
			if err == io.EOF {
				break
			} else {
				return t.Fail(err)
			}
		}

		// Write all of buffer
		for ndt := 0; ndt < ns; {
			if nd, err = t.dst.Write(buf[ndt:ns]); err != nil {
				return t.Fail(err)
			}
			ndt += nd
			atomic.AddInt64(&t.bytesCopied, int64(nd))
		}
	}

	return t.Ok(t.totalSize)
}

func (t *TaskCopy) GetProgress() float64 {
	a := atomic.LoadInt64(&t.bytesCopied)
	b := t.totalSize
	return float64(a) / float64(b)
}
