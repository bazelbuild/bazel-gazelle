//go:build appengine || solaris || (!linux && !darwin && !freebsd && !openbsd && !netbsd)
// +build appengine solaris !linux,!darwin,!freebsd,!openbsd,!netbsd

package fastwalk

import (
	"io/fs"
	"os"
)

// readDir calls fn for each directory entry in dirName.
// It does not descend into directories or follow symlinks.
// If fn returns a non-nil error, readDir returns with that error
// immediately.
func readDir(dirName string, fn func(dirName, entName string, de fs.DirEntry) error) error {
	f, err := os.Open(dirName)
	if err != nil {
		return err
	}
	des, readErr := f.ReadDir(-1)
	f.Close()
	if readErr != nil && len(des) == 0 {
		return readErr
	}

	var skipFiles bool
	for _, d := range des {
		if skipFiles && d.Type().IsRegular() {
			continue
		}
		// Need to use FileMode.Type().Type() for fs.DirEntry
		e := newDirEntry(dirName, d)
		if err := fn(dirName, d.Name(), e); err != nil {
			if err != ErrSkipFiles {
				return err
			}
			skipFiles = true
		}
	}

	return readErr
}
