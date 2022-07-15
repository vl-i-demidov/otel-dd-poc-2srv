package server

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	ddtracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"io/ioutil"
	"log"
	"net/http"
	"otel-dd-poc-2srv/internal/config"
	"otel-dd-poc-2srv/internal/dt/dd"
	oteldt "otel-dd-poc-2srv/internal/dt/otel"
)

// DT book, p. 55 - propagator stack - headers are _injected_ for all registered propagators (OTEL, DD(?), B3),
// _context_ is extracted once it found in any propagator
var globalCfg config.Config

func StartMain(cfg config.Config) {
	globalCfg = cfg

	if cfg.Profiling.Enabled {
		stopProfiling, err := dd.StartProfiling()
		if err != nil {
			log.Fatalln("Couldn't start DD profiling.", err)
		}
		defer stopProfiling()
	}

	var router CustomRouter

	if cfg.Tracing.Protocol == config.TracingDD {
		log.Print("DataDog tracing enabled")

		stopTracing := dd.StartTracing()
		defer stopTracing()

		// Create a traced mux router.
		// if router is created before trace.Start call, serviceName will be overriden
		router = muxtrace.NewRouter()
	} else if cfg.Tracing.Protocol == config.TracingOTEL {
		log.Print("OpenTelemetry tracing enabled")

		stopTraceExporter := oteldt.SetUpOtelTracing(cfg.Tracing)
		defer stopTraceExporter()

		simpleRouter := mux.NewRouter()
		simpleRouter.Use(
			otelmux.Middleware(cfg.Tracing.Service, otelmux.WithTracerProvider(otel.GetTracerProvider())),
		)
		router = simpleRouter
	}

	router.HandleFunc("/ping", pingHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", cfg.HttpPort), router)
}

type CustomRouter interface {
	http.Handler
	HandleFunc(path string, f func(http.ResponseWriter, *http.Request)) *mux.Route
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	frwd := r.URL.Query().Get("forward")
	if frwd == "true" {
		execForwardHandler(globalCfg.ForwardEndpoint, "ping", w, r)
		return
	}
	w.Write([]byte(fmt.Sprintf("PONG from %s", globalCfg.Tracing.Service)))
}

func execForwardHandler(baseUrl, path string, w http.ResponseWriter, r *http.Request) {
	forwardHandlerGeneric(fmt.Sprintf("http://%s/%s", baseUrl, path), w, r)
}

func forwardHandlerGeneric(url string, w http.ResponseWriter, r *http.Request) {
	log.Println("Request is forwarded to", url)

	ctx := r.Context()

	// generate request
	request, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	for k, v := range r.URL.Query() {
		if k != "forward" {
			request.URL.Query().Add(k, v[0])
		}
	}

	// create span
	ctx, span := startSpanItem(ctx)
	defer span.end()

	// inject span context into request
	span.injectContextIntoRequest(ctx, request)

	res, err := http.DefaultClient.Do(request)

	span.enrichWithResponse(res)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%s", string(body))
}

// https://docs.datadoghq.com/tracing/trace_collection/custom_instrumentation/go/
func startSpanItem(ctx context.Context) (context.Context, spanItem) {
	var s spanItem
	var context context.Context
	if globalCfg.Tracing.Protocol == config.TracingDD {
		span, ddctx := ddtracer.StartSpanFromContext(ctx, "outbound.call")
		s = &datadogSpan{span}
		context = ddctx
	} else if globalCfg.Tracing.Protocol == config.TracingOTEL {
		otelctx, span := otel.Tracer("").Start(ctx, "outbound.call", trace.WithSpanKind(trace.SpanKindClient))
		//span.SetAttributes(semconv.HTTPClientAttributesFromHTTPRequest(r)...)
		s = &otelSpan{span}
		context = otelctx
	} else {
		s = &noopSpan{}
		context = ctx
	}

	return context, s
}

type spanItem interface {
	end()
	injectContextIntoRequest(ctx context.Context, r *http.Request)
	enrichWithResponse(resp *http.Response)
}

type datadogSpan struct {
	ddtracer.Span
}

func (s *datadogSpan) end() {
	s.Finish()
}

func (s *datadogSpan) injectContextIntoRequest(ctx context.Context, r *http.Request) {
	ddtracer.Inject(s.Context(), ddtracer.HTTPHeadersCarrier(r.Header))
}

func (s *datadogSpan) enrichWithResponse(resp *http.Response) {
	// noop for now
}

type otelSpan struct {
	trace.Span
}

func (s *otelSpan) end() {
	s.End()
}

func (s *otelSpan) injectContextIntoRequest(ctx context.Context, r *http.Request) {
	context, request := otelhttptrace.W3C(ctx, r) // is this line needed?
	otelhttptrace.Inject(context, request)
}

func (s *otelSpan) enrichWithResponse(resp *http.Response) {
	// noop for now
}

type noopSpan struct{}

func (s *noopSpan) end() {}

func (s *noopSpan) injectContextIntoRequest(ctx context.Context, r *http.Request) {}

func (s *noopSpan) enrichWithResponse(resp *http.Response) {}
