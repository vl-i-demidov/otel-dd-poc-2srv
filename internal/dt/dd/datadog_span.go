package dd

import (
	"context"
	ddtracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"net/http"
)

type DatadogSpan struct {
	ddtracer.Span
}

func (s *DatadogSpan) Stop() {
	s.Finish()
}

// InjectContextIntoRequest based on https://docs.datadoghq.com/tracing/trace_collection/custom_instrumentation/go/
func (s *DatadogSpan) InjectContextIntoRequest(_ context.Context, r *http.Request) {
	err := ddtracer.Inject(s.Context(), ddtracer.HTTPHeadersCarrier(r.Header))
	if err != nil {
		// do nothing
	}
}

func (s *DatadogSpan) EnrichWithResponse(_ *http.Response) {
	// noop for now
}
