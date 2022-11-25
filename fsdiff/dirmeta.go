package fsdiff

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"golang.org/x/exp/slices"
)

const SEP = "/"

type FileMeta struct {
	Name string
	Mode fs.FileMode
	// Sha256 hash of the file contents
	Hash string
	// Size of the file
	Size int64
}

func (fm *FileMeta) IsSame(rhs *FileMeta) bool {
	return *fm == *rhs
}

type DirMeta struct {
	Name  string
	Mode  fs.FileMode
	Files map[string]*FileMeta
	Dirs  map[string]*DirMeta
}

func NewDirMeta(name string) *DirMeta {
	return &DirMeta{
		Name:  name,
		Mode:  fs.ModeDir,
		Files: make(map[string]*FileMeta),
		Dirs:  make(map[string]*DirMeta),
	}
}

// Add an existing FileMeta to this directory
func (d *DirMeta) AddFile(fm *FileMeta) {
	d.Files[fm.Name] = fm
}

// Add an existing DirMeta to this directory
func (d *DirMeta) AddDir(dm *DirMeta) {
	d.Dirs[dm.Name] = dm
}

// Similar to the 'mkdir -p' command this will return a new directory at the
// given path, creating missing directories if necessary.
func (d *DirMeta) MakeDir(fpath string) *DirMeta {
	fpath = path.Clean(fpath)
	if fpath == "." {
		return d
	}
	parts := strings.Split(fpath, SEP)
	dm := d
	for _, name := range parts {
		next := dm.Dirs[name]
		if next == nil {
			next = NewDirMeta(name)
			dm.Dirs[name] = next
		}
		dm = next
	}
	return dm
}

func (d *DirMeta) GetDir(fpath string) *DirMeta {
	fpath = path.Clean(fpath)
	if fpath == "." {
		return d
	}
	parts := strings.Split(fpath, SEP)
	dm := d
	for _, name := range parts {
		if next := dm.Dirs[name]; next == nil {
			return nil
		} else {
			dm = next
		}
	}
	return dm
}

func (d *DirMeta) GetFile(fpath string) *FileMeta {
	parent, name := path.Split(path.Clean(fpath))
	if d1 := d.GetDir(parent); d1 != nil {
		return nil
	} else {
		return d1.Files[name]
	}
}

// Take a pre-split path and get the directory from it
func (d *DirMeta) getSplitDir(path []string) *DirMeta {
	di := d
	for _, name := range path {
		if name == "" || name == "." {
			continue
		}
		next := di.Dirs[name]
		if next == nil {
			return nil
		}
		di = next
	}
	return di
}

// Helper for filling out a DirMeta tree from a filesystem, archive, or other source.
type DirMetaBuilder struct {
	// Root of the file metadata tree
	Root *DirMeta
	// Any *recoverable* errors encountered when trying to read a path will be
	// added here, things like permission errors or read errors. Just because
	// one file can't be read doesn't mean all files can't.
	PathErrors map[string]error
	// An array of absolute paths that will be ignored
	Excludes []string
	// Byte buffer for reading/copying
	buf []byte
}

func NewDirMetaBuilder(root *DirMeta) *DirMetaBuilder {
	return &DirMetaBuilder{
		Root:       root,
		PathErrors: make(map[string]error),
		Excludes:   make([]string, 0),
		buf:        make([]byte, 4096),
	}
}

// Similar to fs.WalkDirFunc but also receives a reader to read the contents of the file or nil
// if the entry is a directory.
func (b *DirMetaBuilder) Add(fpath string, d fs.DirEntry, rd io.Reader, inErr error) error {
	if inErr != nil {
		b.PathErrors[fpath] = inErr
		return nil
	}
	// Skip excludes
	if slices.Contains(b.Excludes, fpath) {
		if d.IsDir() {
			return fs.SkipDir
		} else {
			return nil
		}
	}
	// Get parent directory
	parentPath := path.Dir(fpath)
	parentDir := b.Root.MakeDir(parentPath)
	if parentDir == nil {
		return fmt.Errorf("could not create directory '%s'", parentPath)
	}
	if d.IsDir() {
		dm := NewDirMeta(d.Name())
		dm.Mode = d.Type()
		parentDir.AddDir(dm)
	} else {
		size, hash, err := b.HashReader(rd)
		if err != nil {
			b.PathErrors[fpath] = err
			return nil
		}
		parentDir.AddFile(&FileMeta{
			Name: d.Name(),
			Mode: d.Type(),
			Size: size,
			Hash: hash,
		})
	}
	return nil
}

// Calls fs.WalkDir on fsys and then adds the files and directories using b.Add.
func (b *DirMetaBuilder) AddFs(fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return b.Add(path, d, nil, err)
		} else {
			rd, err := fsys.Open(path)
			if err != nil {
				defer rd.Close()
			}
			return b.Add(path, d, rd, err)
		}
	})
}

// Iterates through the tar file and calls b.Add for each entry.
func (b *DirMetaBuilder) AddTar(tr *tar.Reader) error {
	for {
		header, err := tr.Next()
		// Exit case
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return err
		}
		// Normal case
		info := fs.FileInfoToDirEntry(header.FileInfo())
		var rd io.Reader = tr
		if info.IsDir() {
			rd = nil
		}
		if err := b.Add(header.Name, info, rd, nil); err != nil {
			return err
		}
	}
}

func (b *DirMetaBuilder) HasErrors() bool {
	return len(b.PathErrors) > 0
}

// Hashes the contents of a reader, returning the total size, hash, and any error.
func (b *DirMetaBuilder) HashReader(rd io.Reader) (int64, string, error) {
	h := sha256.New()
	size, err := io.CopyBuffer(h, rd, b.buf)
	if err != nil {
		return 0, "", err
	}
	hs := hex.EncodeToString(h.Sum([]byte{}))
	return size, hs, nil
}
