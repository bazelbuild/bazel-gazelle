//go:build (aix || darwin || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd) && !appengine && !solaris
// +build aix darwin dragonfly freebsd js,wasm linux netbsd openbsd
// +build !appengine
// +build !solaris

package fastwalk

import (
	"io/fs"
	"os"
)

type unixDirent struct {
	parent string
	name   string
	typ    os.FileMode
	info   *fileInfo
	stat   *fileInfo
}

func (d *unixDirent) Name() string      { return d.name }
func (d *unixDirent) IsDir() bool       { return d.typ.IsDir() }
func (d *unixDirent) Type() os.FileMode { return d.typ }

func (d *unixDirent) Info() (fs.FileInfo, error) {
	info := loadFileInfo(&d.info)
	info.once.Do(func() {
		info.FileInfo, info.err = os.Lstat(d.parent + "/" + d.name)
	})
	return info.FileInfo, info.err
}

func (d *unixDirent) Stat() (fs.FileInfo, error) {
	if d.typ&os.ModeSymlink == 0 {
		return d.Info()
	}
	stat := loadFileInfo(&d.stat)
	stat.once.Do(func() {
		stat.FileInfo, stat.err = os.Stat(d.parent + "/" + d.name)
	})
	return stat.FileInfo, stat.err
}

func newUnixDirent(parent, name string, typ os.FileMode) *unixDirent {
	return &unixDirent{
		parent: parent,
		name:   name,
		typ:    typ,
	}
}

func fileInfoToDirEntry(dirname string, fi fs.FileInfo) fs.DirEntry {
	info := &fileInfo{
		FileInfo: fi,
	}
	info.once.Do(func() {})
	return &unixDirent{
		parent: dirname,
		name:   fi.Name(),
		typ:    fi.Mode().Type(),
		info:   info,
	}
}
