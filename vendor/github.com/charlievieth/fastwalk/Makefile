.PHONY: test_build_darwin_arm64
test_build_darwin_arm64:
	GOOS=darwin GOARCH=arm64 go test -c -o /dev/null

.PHONY: test_build_darwin_amd64
test_build_darwin_amd64:
	GOOS=darwin GOARCH=amd64 go test -c -o /dev/null

.PHONY: test_build_linux_arm64
test_build_linux_arm64:
	GOOS=linux GOARCH=arm64 go test -c -o /dev/null

.PHONY: test_build_linux_amd64
test_build_linux_amd64:
	GOOS=linux GOARCH=amd64 go test -c -o /dev/null

.PHONY: test_build_windows_amd64
test_build_windows_amd64:
	GOOS=windows GOARCH=amd64 go test -c -o /dev/null

.PHONY: test_build_freebsd_amd64
test_build_freebsd_amd64:
	GOOS=freebsd GOARCH=amd64 go test -c -o /dev/null

.PHONY: test_build_openbsd_amd64
test_build_openbsd_amd64:
	GOOS=openbsd GOARCH=amd64 go test -c -o /dev/null

.PHONY: test_build_netbsd_amd64
test_build_netbsd_amd64:
	GOOS=netbsd GOARCH=amd64 go test -c -o /dev/null

# Test that we can build fastwalk on multiple platforms
.PHONY: test_build
test_build: test_build_darwin_arm64 test_build_darwin_amd64 \
	test_build_linux_arm64 test_build_linux_amd64 \
	test_build_windows_amd64 test_build_freebsd_amd64 \
	test_build_openbsd_amd64 test_build_netbsd_amd64

.PHONY: test
test: # runs all tests against the package with race detection and coverage percentage
	@go test -race -cover ./...
ifeq "$(shell go env GOOS)" "darwin"
	@go test -tags nogetdirentries -race -cover ./...
endif

.PHONY: quick
quick: # runs all tests without coverage or the race detector
	@go test ./...

.PHONY: bench
bench:
	@go test -run '^$' -bench . -benchmem ./...

.PHONY: bench_comp
bench_comp:
	@go run ./scripts/bench_comp.go

.PHONY: all
all: test test_build
