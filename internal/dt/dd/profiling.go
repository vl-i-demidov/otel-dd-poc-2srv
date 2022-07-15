package dd

import "gopkg.in/DataDog/dd-trace-go.v1/profiler"

// StartProfiling https://docs.datadoghq.com/tracing/profiler/enabling/go/
func StartProfiling() (func(), error) {

	if err := profiler.Start(
		profiler.WithProfileTypes(
			profiler.CPUProfile,
			profiler.HeapProfile,
			profiler.MutexProfile,
			profiler.GoroutineProfile,
		),
	); err != nil {
		return func() {}, err
	}

	return func() {
		profiler.Stop()
	}, nil
}
