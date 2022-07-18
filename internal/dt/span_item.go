package dt

import (
	"context"
	"net/http"
)

type SpanItem interface {
	Stop()
	InjectContextIntoRequest(ctx context.Context, r *http.Request)
	EnrichWithResponse(resp *http.Response)
}

type NoopSpan struct{}

func (s *NoopSpan) Stop() {}

func (s *NoopSpan) InjectContextIntoRequest(_ context.Context, _ *http.Request) {}

func (s *NoopSpan) EnrichWithResponse(_ *http.Response) {}
