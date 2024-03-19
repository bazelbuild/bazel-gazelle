//go:build unix

package pkg

import "golang.org/x/sys/unix"

func PlatformDependentFunction() string {
	home, _ := unix.Getenv("HOME")
	return home
}
