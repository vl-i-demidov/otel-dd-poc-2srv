package server

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"otel-dd-poc-2srv/internal/pkg/monitoring/tracer"

	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"otel-dd-poc-2srv/internal/config"
	"otel-dd-poc-2srv/internal/dt/dd"
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

	// TODO: check what happens if receiver is unavailable
	stopTraceExporter, err := tracer.StartTracing(cfg.Tracing)
	defer stopTraceExporter()

	if err != nil {
		// TODO: should be a warning
		log.Println("Couldn't start tracer")
	}
	router := mux.NewRouter()
	router.Use(
		otelmux.Middleware(cfg.Tracing.Service, otelmux.WithTracerProvider(otel.GetTracerProvider())),
	)

	router.HandleFunc("/ping", pingHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", cfg.HttpPort), router)
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

	var err error
	_, stop := tracer.Trace(r.Context(), "perform.ping",
		attribute.String("time", time.Now().String()))
	defer func() { stop(err) }()

	sleep := rand.Int63n(1000)
	time.Sleep(time.Duration(sleep) * time.Millisecond)

	errMsg := r.URL.Query().Get("error")
	if errMsg != "" {
		err = errors.New(errMsg)

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - %v", err)))
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

	// create span
	var err error
	ctx, stop := tracer.Trace(ctx, "forward.request", attribute.String("forward.url", url))
	defer func() { stop(err) }()

	// generate request
	request, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	query := request.URL.Query()
	for k, v := range r.URL.Query() {
		if k != "forward" {
			query.Add(k, v[0])
		}
	}
	request.URL.RawQuery = query.Encode()

	client := getHttpClient()

	res, err := client.Do(request)

	//span.EnrichWithResponse(res)

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
	var client = &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	return client
}
