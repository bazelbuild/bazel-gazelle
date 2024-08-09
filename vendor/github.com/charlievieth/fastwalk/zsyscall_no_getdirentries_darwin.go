//go:build nogetdirentries && darwin && go1.12
// +build nogetdirentries,darwin,go1.12

package fastwalk

const useGetdirentries = false

func getdirentries(fd int, _ []byte, _ *uintptr) (int, error) {
	panic("NOT IMPLEMENTED")
}
