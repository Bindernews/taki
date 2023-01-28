package taki

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/bindernews/taki/fsdiff"
	"github.com/bindernews/taki/task"
	"github.com/samber/lo"
)

type ServerImpl struct {
	cfg *ServerConfig
	// Generated diff
	fdiff *fsdiff.FsDiff
	// Root of collected metadata
	rootMeta *fsdiff.DirMeta
	// Tar task
	tarTask *TarTask
}

func (s *ServerImpl) GetRoots(req Empty, res *GetRootsRes) (err error) {
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

func (s *ServerImpl) GenerateDiff(req *GenerateDiffReq, res *Empty) (err error) {
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

// Start collecting files into an archive
func (s *ServerImpl) TarStart(req Empty, res *Empty) error {
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

	s.tarTask = &TarTask{
		BaseTask:  task.NewBaseTask(),
		Output:    s.cfg.Output,
		Root:      s.cfg.Root,
		Files:     files,
		FileSizes: sizes,
	}
	go s.tarTask.Run(context.Background())
	return nil
}

// Get the progress of the tar task
func (s *ServerImpl) TarProgress(req Empty, res *float64) error {
	if s.tarTask == nil {
		return ErrTaskNotStarted
	} else {
		*res = s.tarTask.GetProgress()
		return nil
	}
}

func (s *ServerImpl) SetConfig(config *ServerConfig, res *bool) error {
	s.cfg = config
	*res = true
	return nil
}
