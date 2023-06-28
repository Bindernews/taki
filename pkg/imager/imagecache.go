package imager

import (
	"archive/tar"
	"os"
	"path"
	"sync"

	"github.com/bindernews/taki/pkg/fsdiff"
	"github.com/bindernews/taki/pkg/task"
)

// Caches DirMeta objects for base images (tar files).
// This way in a batch-processing scenario we're not re-reading
// the same tar file mulitple times.
type ImageCache struct {
	// Lock for the whole thing
	lck sync.Mutex
	// Cache of metadata, both in-progress and completed. In-progress requests
	// will have Meta = nil, while finished ones will have Meta set.
	cache map[string]*ImageRequest
}

func (ic *ImageCache) Request(fpath string) *ImageRequest {
	ic.lck.Lock()
	defer ic.lck.Unlock()
	cleanPath := path.Clean(fpath)
	// Get or create the request for the given path
	req := ic.cache[cleanPath]
	if req == nil {
		req = &ImageRequest{
			BaseTask: task.NewBaseTask(),
			Path:     cleanPath,
		}
		ic.cache[cleanPath] = req
		go ic.imageGather(req)
	}
	return req
}

// The metadata gathering process for the ImageRequest, should be run in a goroutine.
func (ic *ImageCache) imageGather(req *ImageRequest) any {
	// Create new builder and meta
	b := fsdiff.NewDirMetaBuilder(fsdiff.NewDirMeta(""))
	file, err := os.Open(req.Path)
	if err != nil {
		return req.Fail(err)
	}
	defer file.Close()

	tr := tar.NewReader(file)
	if err := b.AddTar(tr); err != nil {
		return req.Fail(err)
	}
	// Success!
	return req.Ok(b.Root)
}

type ImageRequest struct {
	*task.BaseTask
	// Resolved path
	Path string
}

func (ir *ImageRequest) Value() *fsdiff.DirMeta {
	return ir.BaseTask.Value().(*fsdiff.DirMeta)
}
