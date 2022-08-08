package tracer

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
	ddtags "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"otel-dd-poc-2srv/internal/config"
)

// StartTracing initializes TraceProvider
func StartTracing(cfg config.Tracing) (stop func(), err error) {

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
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.Service),
			// datadog
			attribute.String(ddtags.Environment, cfg.Environment),
			attribute.String(ddtags.ServiceName, cfg.Service),
			attribute.String(ddtags.Version, cfg.AppVersion),
		))
	if err != nil {
		return func() {}, err
	}

	tracerProvider := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.ParentBased(tracesdk.TraceIDRatioBased(cfg.SamplingRatio))),
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

func Trace(ctx context.Context, name string, attributes ...attribute.KeyValue) (context.Context, func(error)) {

	ctx, span := otel.Tracer("").Start(ctx, name, trace.WithAttributes(attributes...))
	span.SetAttributes(
		append(
			attributes,
			// add Datadog specific attributes
			// TODO do we need all of them for APM to work correctly?
			// OTEL doesn't have a specific resource tag, while DD requires some resource
			// DD even have span_name_as_resource_name in their exporter
			attribute.String(ddtags.ResourceName, name),
			attribute.String(ddtags.SpanName, name),
			attribute.String(ddtags.SpanType, "function"),
		)...,
	)

	return ctx, func(err error) {
		defer span.End()

		if err != nil {
			span.RecordError(err) // TODO: do we need that?
			span.SetStatus(otelcodes.Error, err.Error())
			return
		}

		// TODO: propagate context attributes - may be all custom attributes should go to the context first and
		// only then - to the span. So you put tags not on span, but in context. This way it can be shared later
		// TODO: propagate traceId to logs
	}
}

// ManualDropDatadog forces this span and all its child spans to be dropped.
// Note that it only works when traces are sent to Datadog Agent and
// TraceIDRatioBased sampling ratio is set to 1
func ManualDropDatadog(span trace.Span) {
	span.SetAttributes(
		attribute.Bool(ddtags.ManualDrop, true))
}

// ManualKeepDatadog forces this span and all its child spans to be kept.
// Note that it only works when traces are sent to Datadog Agent and
// TraceIDRatioBased sampling ratio is set to 1
func ManualKeepDatadog(span trace.Span) {
	span.SetAttributes(
		attribute.Bool(ddtags.ManualKeep, true))
}
