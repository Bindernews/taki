package task

// Denotes a function that will always return nil, but
// has a return for implementor convenience.
type Void any

type Task interface {
	Value() any
	// Done returns a channel that is closed when the task is complete, either
	// with a result or an error.
	Done() <-chan struct{}
	//
	Err() error
}

type Progressive interface {
	GetProgress() float64
}

// Simple implementation of Task that can be used as a base
// for more
type BaseTask struct {
	// Done channel
	doneC chan struct{}
	// Error value
	errV error
	// Okay value
	okV any
}

func NewBaseTask() *BaseTask {
	return &BaseTask{
		doneC: make(chan struct{}),
		errV:  nil,
		okV:   nil,
	}
}

func (t *BaseTask) Ok(v any) Void {
	t.okV = v
	close(t.doneC)
	return nil
}

func (t *BaseTask) Fail(err error) Void {
	t.errV = err
	close(t.doneC)
	return nil
}

func (t *BaseTask) Done() <-chan struct{} {
	return t.doneC
}
func (t *BaseTask) Err() error {
	return t.errV
}
func (t *BaseTask) Value() any {
	return t.okV
}
