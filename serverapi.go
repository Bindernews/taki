package taki

import (
	"errors"

	"github.com/bindernews/taki/fsdiff"
)

// Line that is printed prior to switching to binary encoding
const SERVER_START_LINE = "--TAKI SERVER START--"

// Error returned by task APIs when the task isn't found
var ErrTaskNotExist = errors.New("task does not exist")

// Error when a task does not implement the Progressive interface
var ErrTaskNotProgressive = errors.New("task does not implement Progressive")

// Handle for server tasks
type TaskHandle uint

type ServerApi interface {
	GetRoots(req *GetRootsReq, res *GetRootsRes) error
	SetConfig(config *ServerConfig, res *Empty) error
	GenerateDiff(req *GenerateDiffReq, res *GenerateDiffRes) error
	// Start collecting files into an archive
	CollectFilesStart(req Empty, handle *TaskHandle) error
	// Poll the progress of a task, may take a while to return
	TaskProgress(handle TaskHandle, progress *float64) error
}

type Empty struct{}

type GetRootsReq struct{}
type GetRootsRes struct {
	Roots []string
}

type GenerateDiffReq struct {
	Base *fsdiff.DirMeta
}
type GenerateDiffRes struct{}

type CollectFilesRes struct {
	// Number of bytes collected
	Bytes int64
	// Total number of bytes to collect
	Total int
	// Is the task fully complete
	Done bool
}

type ServerConfig struct {
	// Root path to collect and diff from
	Root string
	// List of exclusions (relative to root)
	Exclude []string
	// Output path for CollectFiles
	Output string
}
