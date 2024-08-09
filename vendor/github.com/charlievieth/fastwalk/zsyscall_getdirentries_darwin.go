//go:build !nogetdirentries && darwin && go1.12
// +build !nogetdirentries,darwin,go1.12

package fastwalk

import (
	"syscall"
	"unsafe"
)

const useGetdirentries = true

// Implemented in the runtime package (runtime/sys_darwin.go)
func syscall_syscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)

//go:linkname syscall_syscall6 syscall.syscall6

// Single-word zero for use when we need a valid pointer to 0 bytes.
var _zero uintptr

func getdirentries(fd int, buf []byte, basep *uintptr) (n int, err error) {
	var _p0 unsafe.Pointer
	if len(buf) > 0 {
		_p0 = unsafe.Pointer(&buf[0])
	} else {
		_p0 = unsafe.Pointer(&_zero)
	}
	r0, _, e1 := syscall_syscall6(libc___getdirentries64_trampoline_addr,
		uintptr(fd), uintptr(_p0), uintptr(len(buf)), uintptr(unsafe.Pointer(basep)),
		0, 0)
	n = int(r0)
	if e1 != 0 {
		err = errnoErr(e1)
	} else if n < 0 {
		err = errnoErr(syscall.EINVAL)
	}
	return
}

var libc___getdirentries64_trampoline_addr uintptr

//go:cgo_import_dynamic libc___getdirentries64 __getdirentries64 "/usr/lib/libSystem.B.dylib"
