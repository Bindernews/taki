package fsdiff

import (
	"path"

	"github.com/samber/lo"
)

// A filesystem-diff, determines a full list of files that have been
// added, modified, and removed.
type FsDiff struct {
	// Full paths of files that were added
	Added []string
	// Full paths of files that were removed
	Removed []string
	// Full paths of files that were modified
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

// Compare two directory metadata objects, building a full diff of them.
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

// Adds all files and directories in the given root recursively to self.
func (d *FsDiff) recAdded(root string, dm *DirMeta) {
	// Track files
	addedFiles := lo.Keys(dm.Files)
	prependDir(addedFiles, root)
	d.Added = append(d.Added, addedFiles...)
	// Recurse into directories
	for _, dir := range dm.Dirs {
		d.recAdded(path.Join(root, dir.Name), dir)
	}
}

// Lists all files and directories recursively in 'dm' as being removed in self.
func (d *FsDiff) recRemoved(root string, dm *DirMeta) {
	// Track files
	removedFiles := lo.Keys(dm.Files)
	prependDir(removedFiles, root)
	d.Removed = append(d.Removed, removedFiles...)
	// Recurse into directories
	for _, dir := range dm.Dirs {
		d.recRemoved(path.Join(root, dir.Name), dir)
	}
}

// Helper function to prepend 'dir' to each value in 'files'.
// This modifies 'files'.
func prependDir(files []string, dir string) {
	for i, n := range files {
		files[i] = path.Join(dir, n)
	}
}
