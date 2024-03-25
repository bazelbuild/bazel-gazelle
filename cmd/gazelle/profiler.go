package main

import (
	"os"
	"runtime"
	"runtime/pprof"

	"golang.org/x/exp/trace"
)

type profiler struct {
	fr           *trace.FlightRecorder
	traceProfile string
	cpuProfile   *os.File
	memProfile   string
}

// newProfiler creates a profiler that writes to the given files.
// it returns an empty profiler if both files are empty.
// so that stop() will never fail.
func newProfiler(cpuProfile, memProfile, traceProfile string) (profiler, error) {
	var p profiler
	if traceProfile != "" {
		p.traceProfile = traceProfile
		p.fr = trace.NewFlightRecorder()
		p.fr.Start()
	}

	if cpuProfile == "" {
		p.memProfile = memProfile
		return p, nil
	}

	f, err := os.Create(cpuProfile)
	if err != nil {
		return profiler{}, err
	}
	pprof.StartCPUProfile(f)

	p.cpuProfile = f
	return p, nil
}

func (p *profiler) stop() error {
	if p.fr != nil {
		f, err := os.OpenFile(p.traceProfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
		if err != nil {
			return err
		}
		if _, err := p.fr.WriteTo(f); err != nil {
			return err
		}
		p.fr.Stop()
	}
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
