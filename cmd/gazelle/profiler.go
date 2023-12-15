package main

import (
	"os"
	"runtime"
	"runtime/pprof"
)

type profiler struct {
	cpuProfile *os.File
	memProfile string
}

// newProfiler creates a profiler that writes to the given files.
// it returns an empty profiler if both files are empty.
// so that stop() will never fail.
func newProfiler(cpuProfile, memProfile string) (profiler, error) {
	if cpuProfile == "" {
		return profiler{
			memProfile: memProfile,
		}, nil
	}

	f, err := os.Create(cpuProfile)
	if err != nil {
		return profiler{}, err
	}
	pprof.StartCPUProfile(f)

	return profiler{
		cpuProfile: f,
		memProfile: memProfile,
	}, nil
}

func (p *profiler) stop() error {
	if p.cpuProfile != nil {
		pprof.StopCPUProfile()
		p.cpuProfile.Close()
	}

	if p.memProfile == "" {
		return nil
	}

	f, err := os.Create(p.memProfile)
	if err != nil {
		return err
	}
	defer f.Close()
	runtime.GC()
	return pprof.WriteHeapProfile(f)
}
