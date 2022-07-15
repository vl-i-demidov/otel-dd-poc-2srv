package dd

import "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

// StartTracing starts collecting traces and send them to the DataDog agent.
// Returns a function, which needs to be called before exiting application.
// It relies on three main env variables: DD_ENV, DD_SERVICE and DD_VERSION.
// https://docs.datadoghq.com/tracing/setup_overview/setup/go/?tab=containers
func StartTracing() func() {

	tracer.Start()
	return func() {
		tracer.Stop()
	}
}
