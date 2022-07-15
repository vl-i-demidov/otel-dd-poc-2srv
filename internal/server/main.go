package server

import (
	"context"
	"fmt"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"otel-dd-poc-2srv/internal/config"
	"otel-dd-poc-2srv/internal/dt/dd"
	oteldt "otel-dd-poc-2srv/internal/dt/otel"
)

var globalCfg config.Config

func StartMain(cfg config.Config) {
	globalCfg = cfg

	log.Println(cfg.HttpPort)

	if cfg.Profiling.Enabled {
		stopProfiling, err := dd.StartProfiling()
		if err != nil {
			log.Fatalln("Couldn't start DD profiling.", err)
		}
		defer stopProfiling()
	}

	// Create a traced mux router.
	mux := muxtrace.NewRouter()

	if cfg.Tracing.Protocol == config.TracingDD {
		stopTracing := dd.StartTracing()
		defer stopTracing()
	} else if cfg.Tracing.Protocol == config.TracingOTEL {
		stopTraceExporter := oteldt.SetUpOtelTracing(cfg.Tracing)
		defer stopTraceExporter()

		mux.Use(
			otelmux.Middleware(cfg.Tracing.Service, otelmux.WithTracerProvider(otel.GetTracerProvider())),
		)
	}

	// Continue using the router as you normally would.
	mux.HandleFunc("/ping", pingHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", cfg.HttpPort), mux)
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
	forwardHandler(fmt.Sprintf("http://%s/%s", baseUrl, path), w, r)
}
func forwardHandler(url string, w http.ResponseWriter, r *http.Request) {
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
	ctx, span := startSpan(ctx, r)
	defer span.End()

	// inject span context into request
	injectSpanContext(ctx, request)

	client := http.DefaultClient
	res, err := client.Do(request)

	enrichSpan(span, res)

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

func startSpan(ctx context.Context, r *http.Request) (context.Context, trace.Span) {
	context, span := otel.Tracer("").Start(ctx, "outbound.call", trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(semconv.HTTPClientAttributesFromHTTPRequest(r)...)
	return context, span
}

func injectSpanContext(ctx context.Context, r *http.Request) (context.Context, *http.Request) {
	context, request := otelhttptrace.W3C(ctx, r) // is this line needed?
	otelhttptrace.Inject(context, request)
	return context, request
}

func enrichSpan(span trace.Span, resp *http.Response) {
	span.SetAttributes(attribute.Int("outbound.status_code", resp.StatusCode))
	span.SetStatus(semconv.SpanStatusFromHTTPStatusCode(resp.StatusCode))
}
