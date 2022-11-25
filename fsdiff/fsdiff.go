package fsdiff

import (
	"path"

	"github.com/samber/lo"
)

type FsDiff struct {
	Added    []string
	Removed  []string
	Modified []string
}

func NewFsDiff() *FsDiff {
	return &FsDiff{}
}

// Returns the combied list of added and modified files
func (d *FsDiff) GetAddedModified() []string {
	lst := make([]string, 0, len(d.Added)+len(d.Modified))
	lst = append(lst, d.Added...)
	lst = append(lst, d.Modified...)
	return lst
}

func (d *FsDiff) Compare(lt *DirMeta, rt *DirMeta) error {
	return d.compareDirs("", lt, rt)
}

func (d *FsDiff) compareDirs(root string, lt *DirMeta, rt *DirMeta) error {
	// First diff files
	ltFiles := lo.Keys(lt.Files)
	rtFiles := lo.Keys(rt.Files)
	filesDeleted, filesAdded := lo.Difference(ltFiles, rtFiles)
	filesSame := lo.Intersect(ltFiles, rtFiles)

	// Add directly added/removed files
	prependDir(filesAdded, root)
	prependDir(filesDeleted, root)
	d.Added = append(d.Added, filesAdded...)
	d.Removed = append(d.Removed, filesDeleted...)
	// Check same files
	for _, name := range filesSame {
		ltF := lt.Files[name]
		rtF := rt.Files[name]
		if !ltF.IsSame(rtF) {
			d.Modified = append(d.Modified, path.Join(root, name))
		}
	}

	// Diff directories
	ltDirs := lo.Keys(lt.Dirs)
	rtDirs := lo.Keys(rt.Dirs)
	dirsDeleted, dirsAdded := lo.Difference(ltDirs, rtDirs)
	dirsSame := lo.Intersect(ltDirs, rtDirs)

	// Recursively track added/removed directories
	for _, name := range dirsAdded {
		d.recAdded(path.Join(root, name), rt.Dirs[name])
	}
	for _, name := range dirsDeleted {
		d.recRemoved(path.Join(root, name), lt.Dirs[name])
	}
	// Recursively handle directories that are the same
	for _, name := range dirsSame {
		ltD := lt.Dirs[name]
		rtD := rt.Dirs[name]
		if err := d.compareDirs(path.Join(root, name), ltD, rtD); err != nil {
			return err
		}
	}
	return nil
}

func (d *FsDiff) recAdded(root string, dm *DirMeta) {
	// TODO
}

func (d *FsDiff) recRemoved(root string, dm *DirMeta) {
	// TODO
}

func prependDir(files []string, dir string) {
	for i, n := range files {
		files[i] = path.Join(dir, n)
	}
}

// func (d *FsDiff)
