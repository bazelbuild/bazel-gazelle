package fastwalk

import (
	"io/fs"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"
)

type fileInfo struct {
	once sync.Once
	fs.FileInfo
	err error
}

func loadFileInfo(pinfo **fileInfo) *fileInfo {
	ptr := (*unsafe.Pointer)(unsafe.Pointer(pinfo))
	fi := (*fileInfo)(atomic.LoadPointer(ptr))
	if fi == nil {
		fi = &fileInfo{}
		if !atomic.CompareAndSwapPointer(
			(*unsafe.Pointer)(unsafe.Pointer(pinfo)), // adrr
			nil,                                      // old
			unsafe.Pointer(fi),                       // new
		) {
			fi = (*fileInfo)(atomic.LoadPointer(ptr))
		}
	}
	return fi
}

// StatDirEntry returns the fs.FileInfo for the file or subdirectory described
// by the entry.  If the entry is a symbolic link, StatDirEntry returns the
// fs.FileInfo for the file the line references (os.Stat).
// If fs.DirEntry de is a fastwalk.DirEntry it's Stat() method is used and the
// returned fs.FileInfo may be a previously cached result.
func StatDirEntry(path string, de fs.DirEntry) (fs.FileInfo, error) {
	if de.Type()&os.ModeSymlink == 0 {
		return de.Info()
	}
	if d, ok := de.(DirEntry); ok {
		return d.Stat()
	}
	return os.Stat(path)
}
