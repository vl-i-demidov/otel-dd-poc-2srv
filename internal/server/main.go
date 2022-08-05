package server

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	//"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	ddhttp "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	ddtracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"otel-dd-poc-2srv/internal/config"
	"otel-dd-poc-2srv/internal/dt"
	"otel-dd-poc-2srv/internal/dt/dd"
	oteldt "otel-dd-poc-2srv/internal/dt/otel"
	"time"
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
	for k, v := range r.URL.Query() {
		if k != "forward" {
			log.Println(k, "=", v)
		}
	}
	frwd := r.URL.Query().Get("forward")
	if frwd == "true" {
		execForwardHandler(globalCfg.ForwardEndpoint, "ping", w, r)
		return
	}

	sleep := rand.Int63n(1000)
	time.Sleep(time.Duration(sleep) * time.Millisecond)

	err := r.URL.Query().Get("error")
	if err == "true" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Intentional error happened"))
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

	query := request.URL.Query()
	for k, v := range r.URL.Query() {
		if k != "forward" {
			query.Add(k, v[0])
		}
	}
	request.URL.RawQuery = query.Encode()

	// create span
	ctx, span := startSpanItem(ctx)
	defer span.Stop()

	client := getHttpClient()

	res, err := client.Do(request)

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

func getHttpClient() *http.Client {
	var client *http.Client
	if globalCfg.Tracing.Protocol == config.TracingDD {
		client = ddhttp.WrapClient(&http.Client{})
	} else if globalCfg.Tracing.Protocol == config.TracingOTEL {
		client = &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	} else {
		client = http.DefaultClient
	}
	return client
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
