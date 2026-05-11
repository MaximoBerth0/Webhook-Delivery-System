package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace/noop"
)

func SetupTelemetry(ctx context.Context) (func(context.Context) error, error) {
	if !isOtelEnabled() {
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func(context.Context) error { return nil }, nil
	}

	// get configuration from environment
	serviceName := getEnv("SERVICE_NAME", "webhook-delivery-system")
	serviceVersion := getEnv("SERVICE_VERSION", "unknown")
	environment := getEnv("ENVIRONMENT", "development")
	exporterURL := getEnv("OTEL_EXPORTER_URL", "http://localhost:4318")

	// create exporter with custom endpoint
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(parseEndpoint(exporterURL)),
		otlptracehttp.WithInsecure(), // replace with TLS config
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	// create resource with all attributes
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			semconv.DeploymentEnvironment(environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	// Create tracer provider with environment-specific sampler
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(getSampler(environment)),
	)

	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

func isOtelEnabled() bool {
	enabled := strings.ToLower(getEnv("OTEL_ENABLED", "true"))
	return enabled == "true" || enabled == "1" || enabled == "yes"
}

func getSampler(environment string) sdktrace.Sampler {
	env := strings.ToLower(environment)

	switch env {
	case "production", "prod":
		// sample 10% to reduce costs
		return sdktrace.TraceIDRatioBased(0.1)
	case "staging":
		// sample 50% in staging
		return sdktrace.TraceIDRatioBased(0.5)
	default:
		// sample everything in development
		return sdktrace.AlwaysSample()
	}
}

func parseEndpoint(url string) string {
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	return url
}
