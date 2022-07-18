package otel

import (
	"context"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel/trace"
	"net/http"
)

type OtelSpan struct {
	trace.Span
}

func (s *OtelSpan) Stop() {
	s.End()
}

func (s *OtelSpan) InjectContextIntoRequest(ctx context.Context, r *http.Request) {
	w3cCtx, request := otelhttptrace.W3C(ctx, r) // is this line needed?
	otelhttptrace.Inject(w3cCtx, request)
}

func (s *OtelSpan) EnrichWithResponse(_ *http.Response) {
	// noop for now
}
