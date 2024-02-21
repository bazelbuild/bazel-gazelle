//go:build darwin && go1.12
// +build darwin,go1.12

package fastwalk

import (
	"syscall"
	"unsafe"
)

// Implemented in the runtime package (runtime/sys_darwin.go)
func syscall_syscall(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)
func syscall_syscallPtr(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)

//go:linkname syscall_syscall syscall.syscall
//go:linkname syscall_syscallPtr syscall.syscallPtr

func closedir(dir uintptr) (err error) {
	_, _, e1 := syscall_syscall(libc_closedir_trampoline_addr, uintptr(dir), 0, 0)
	if e1 != 0 {
		err = e1
	}
	return
}

var libc_closedir_trampoline_addr uintptr

//go:cgo_import_dynamic libc_closedir closedir "/usr/lib/libSystem.B.dylib"

func readdir_r(dir uintptr, entry *syscall.Dirent, result **syscall.Dirent) syscall.Errno {
	res, _, _ := syscall_syscall(libc_readdir_r_trampoline_addr, uintptr(dir), uintptr(unsafe.Pointer(entry)), uintptr(unsafe.Pointer(result)))
	return syscall.Errno(res)
}

var libc_readdir_r_trampoline_addr uintptr

//go:cgo_import_dynamic libc_readdir_r readdir_r "/usr/lib/libSystem.B.dylib"

func opendir(path string) (dir uintptr, err error) {
	// We implent opendir so that we don't have to open a file, duplicate
	// it's FD, then call fdopendir with it.
	p, err := syscall.BytePtrFromString(path)
	if err != nil {
		return 0, err
	}
	r0, _, e1 := syscall_syscallPtr(libc_opendir_trampoline_addr, uintptr(unsafe.Pointer(p)), 0, 0)
	dir = uintptr(r0)
	if e1 != 0 {
		err = e1
	}
	return dir, err
}

var libc_opendir_trampoline_addr uintptr

//go:cgo_import_dynamic libc_opendir opendir "/usr/lib/libSystem.B.dylib"
