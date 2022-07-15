package otel

import (
	"context"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"log"
	"otel-dd-poc-2srv/internal/config"
	"time"
)

func SetUpOtelTracing(cfg config.Tracing) (stop func()) {

	// create low-level client to export tracing
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(cfg.ReceiverEndpoint),
		otlptracegrpc.WithDialOption(grpc.WithBlock()))

	// create high-level exporting client
	// In real life we should connect in background
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	log.Print("Connecting to trace receiver......")

	traceExp, err := otlptrace.New(ctx, client)
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Connected!")

	// tags can be added here
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String(ext.Environment, cfg.Environment),
			attribute.String(ext.ServiceName, cfg.Service),
			attribute.String(ext.Version, cfg.AppVersion),
		))

	tracerProvider := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		tracesdk.WithBatcher(traceExp),
		tracesdk.WithResource(res),
	)

	// create OTEL+B3 propagators to handle (inject/extract) multiple tracing headers formats
	// OTEL supports B3 format https://opentelemetry.io/docs/instrumentation/go/manual/#propagators-and-context
	// datadog supports B3 format https://docs.datadoghq.com/tracing/trace_collection/custom_instrumentation/go/#b3-headers-extraction-and-injection
	compositePropagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, b3.New())
	otel.SetTextMapPropagator(compositePropagator)
	// set global trace provider
	otel.SetTracerProvider(tracerProvider)

	return func() {
		cxt, cancel := context.WithTimeout(ctx, 50*time.Second)
		defer cancel()

		if err := traceExp.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
	}
}
