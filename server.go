package taki

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/rpc"
	"os"

	"github.com/bindernews/taki/fsdiff"
	"github.com/bindernews/taki/task"
	"github.com/samber/lo"
)

type ServerImpl struct {
	cfg   *ServerConfig
	fdiff *fsdiff.FsDiff
	// Root of collected metadata
	rootMeta *fsdiff.DirMeta
	// Next task handle
	nextTaskHandle TaskHandle
	// Map of async tasks
	tasks map[TaskHandle]task.Task
}

func (s *ServerImpl) GetRoots(req *GetRootsReq, res *GetRootsRes) (err error) {
	// Init vars and return values
	var rootInode, curInode uint
	var procItems []os.DirEntry
	res.Roots = make([]string, 0)

	if rootInode, err = GetInode("/"); err != nil {
		return
	}
	if procItems, err = os.ReadDir("/proc/"); err != nil {
		return
	}
	for _, pid := range procItems {
		if !pid.IsDir() {
			continue
		}
		rootPath := fmt.Sprintf("/proc/%s/root", pid.Name())
		curInode, err = GetInode(rootPath)
		// If not found, ignore
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return
		}
		// If the inodes are different, it's a new root
		if curInode != rootInode {
			res.Roots = append(res.Roots, rootPath)
		}
	}
	return nil
}

func (s *ServerImpl) GenerateDiff(req *GenerateDiffReq, res *GenerateDiffRes) (err error) {
	if s.cfg == nil {
		return errors.New("config is not set")
	}

	s.rootMeta = fsdiff.NewDirMeta("")
	b := fsdiff.NewDirMetaBuilder(s.rootMeta)
	if err = b.AddFs(os.DirFS(s.cfg.Root)); err != nil {
		return err
	}
	s.fdiff = fsdiff.NewFsDiff()
	if err = s.fdiff.Compare(req.Base, s.rootMeta); err != nil {
		return
	}
	return
}

func (s *ServerImpl) CollectFilesStart(req Empty, handle *TaskHandle) error {
	// Get the files and their corresponding sizes
	files := s.fdiff.GetAddedModified()
	sizes := lo.Map(files, func(path string, _ int) int64 {
		fm := s.rootMeta.GetFile(path)
		if fm == nil {
			return 0
		} else {
			return fm.Size
		}
	})

	tt := &TarTask{
		BaseTask:  task.NewBaseTask(),
		Output:    s.cfg.Output,
		Root:      s.cfg.Root,
		Files:     files,
		FileSizes: sizes,
	}
	// Add and start the task
	*handle = s.addTask(tt)
	go tt.Run(context.Background())
	return nil
}

func (s *ServerImpl) TaskProgress(handle TaskHandle, res *float64) error {
	tt := s.tasks[handle]
	if tt == nil {
		return ErrTaskNotExist
	}
	// TODO check if task is done and if so return error if task responded with error
	if prog, ok := s.tasks[handle].(task.Progressive); ok {
		*res = prog.GetProgress()
		return nil
	} else {
		return ErrTaskNotProgressive
	}
}

func (s *ServerImpl) SetConfig(config *ServerConfig, res *bool) error {
	s.cfg = config
	*res = true
	return nil
}

func (s *ServerImpl) addTask(t task.Task) TaskHandle {
	h := s.nextTaskHandle
	s.nextTaskHandle += 1
	s.tasks[h] = t
	return h
}

func main() {
	fmt.Println(SERVER_START_LINE)
	conn := &StdioRw{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}
	rpc.RegisterName(RPC_FILE_CLS, NewRpcFile())
	rpc.RegisterName("ServerApi", &ServerImpl{})

	go rpc.ServeConn(conn)
}

type StdioRw struct {
	io.Reader
	io.Writer
}

func (rw *StdioRw) Close() error {
	return nil
}
