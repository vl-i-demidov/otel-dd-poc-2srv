package tracer

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	ddtags "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"otel-dd-poc-2srv/internal/config"
)

// StartTracing initializes TraceProvider
func StartTracing(cfg config.Tracing2) (stop func(), err error) {

	// create gRPC client to export spans
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(cfg.ReceiverEndpoint))

	// TODO: add WithBlock anyway to catch error? but log warn in case of failure?

	// create span exporter
	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return func() {}, err
	}

	// tags can be added here
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			// datadog
			attribute.String(ddtags.Environment, cfg.Environment),
			attribute.String(ddtags.ServiceName, cfg.Service),
			attribute.String(ddtags.Version, cfg.AppVersion),
			// TODO: use also OTEL standards

		))
	if err != nil {
		return func() {}, err
	}

	tracerProvider := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()), // TODO: what about sampling
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(res),
	)

	// set propagator (injects and extracts trace headers)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	// set global trace provider
	otel.SetTracerProvider(tracerProvider)

	return func() {
		err := tracerProvider.Shutdown(context.Background())
		if err != nil {
			// TODO: log error
		}
	}, nil
}

func Trace(ctx context.Context, name string, opts ...tracer.StartSpanOption) (context.Context, func(error)) {

	ctx, span := otel.Tracer("").Start(ctx, name, trace.WithSpanKind(trace.SpanKindInternal))

	// add Datadog specific attributes
	// TODO do we need all of them for APM to work correctly?
	span.SetAttributes(
		// OTEL doesn't have a specific resource tag, while DD requires some resource
		// DD even have span_name_as_resource_name in their exporter
		attribute.String(ddtags.ResourceName, name),
		attribute.String(ddtags.SpanName, name),
		attribute.String(ddtags.SpanType, "function"))

	return ctx, func(err error) {

		// TODO: add error to span
		//defer span.Finish(tracer.WithError(err))

		//propagateAttributes(ctx, span)
	}
}
