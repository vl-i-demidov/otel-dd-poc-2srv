package server

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	ddtracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"io/ioutil"
	"log"
	"net/http"
	"otel-dd-poc-2srv/internal/config"
	"otel-dd-poc-2srv/internal/dt"
	"otel-dd-poc-2srv/internal/dt/dd"
	oteldt "otel-dd-poc-2srv/internal/dt/otel"
)

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
	defer span.Stop()

	// inject span context into request
	span.InjectContextIntoRequest(ctx, request)

	res, err := http.DefaultClient.Do(request)

	span.EnrichWithResponse(res)

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

func startSpanItem(ctx context.Context) (context.Context, dt.SpanItem) {
	var s dt.SpanItem
	var resCtx context.Context
	if globalCfg.Tracing.Protocol == config.TracingDD {
		span, ddctx := ddtracer.StartSpanFromContext(ctx, "outbound.call")
		s = &dd.DatadogSpan{Span: span}
		resCtx = ddctx
	} else if globalCfg.Tracing.Protocol == config.TracingOTEL {
		otelctx, span := otel.Tracer("").Start(ctx, "outbound.call", trace.WithSpanKind(trace.SpanKindClient))
		//span.SetAttributes(semconv.HTTPClientAttributesFromHTTPRequest(r)...)
		s = &oteldt.OtelSpan{Span: span}
		resCtx = otelctx
	} else {
		s = &dt.NoopSpan{}
		resCtx = ctx
	}

	return resCtx, s
}
