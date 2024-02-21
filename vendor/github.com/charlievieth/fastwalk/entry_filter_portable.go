//go:build appengine || (!linux && !darwin && !freebsd && !openbsd && !netbsd && !windows && !solaris)
// +build appengine !linux,!darwin,!freebsd,!openbsd,!netbsd,!windows,!solaris

package fastwalk

import (
	"io/fs"
	"path/filepath"
	"sync"
)

type EntryFilter struct {
	// we assume most files have not been seen so
	// no need for a RWMutex
	mu   sync.Mutex
	seen map[string]struct{}
}

func (e *EntryFilter) Entry(path string, _ fs.DirEntry) bool {
	name, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	e.mu.Lock()
	if e.seen == nil {
		e.seen = make(map[string]struct{}, 128)
	}
	_, ok := e.seen[name]
	if !ok {
		e.seen[name] = struct{}{}
	}
	e.mu.Unlock()
	return ok
}

func NewEntryFilter() *EntryFilter {
	return &EntryFilter{seen: make(map[string]struct{}, 128)}
}
