//go:build windows

package pkg

func PlatformDependentFunction() string {
	return "C:\\Users\\gopher"
}
