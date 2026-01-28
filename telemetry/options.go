package telemetry

import (
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/oauth2/google"
)

type config struct {
	otelToCloud bool

	// optional credentials TODO add implementation
	credentials *google.Credentials
	// TODO resource
	resource            *resource.Resource
	spanProcessors      []sdktrace.SpanProcessor
	metricReaders       []sdkmetric.Reader
	logRecordProcessors []sdklog.Processor

	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	loggerProvider *sdklog.LoggerProvider
}

// Option configures a adk telemetry.
type Option interface {
	apply(*config) error
}

type optionFunc func(*config) error

func (fn optionFunc) apply(cfg *config) error {
	return fn(cfg)
}

// WithOtelToCloud enables exporting telemetry to Cloud
func WithOtelToCloud() Option {
	return optionFunc(func(cfg *config) error {
		cfg.otelToCloud = true
		return nil
	})
}

func WithResource(r *resource.Resource) Option {
	return optionFunc(func(cfg *config) error {
		cfg.resource = r
		return nil
	})
}

func WithCredentials(c *google.Credentials) Option {
	return optionFunc(func(cfg *config) error {
		cfg.credentials = c
		return nil
	})
}

func WithSpanProcessors(p ...sdktrace.SpanProcessor) Option {
	return optionFunc(func(cfg *config) error {
		cfg.spanProcessors = append(cfg.spanProcessors, p...)
		return nil
	})
}

func WithMetricReaders(r ...sdkmetric.Reader) Option {
	return optionFunc(func(cfg *config) error {
		cfg.metricReaders = append(cfg.metricReaders, r...)
		return nil
	})
}

func WithLogRecordProcessors(p ...sdklog.Processor) Option {
	return optionFunc(func(cfg *config) error {
		cfg.logRecordProcessors = append(cfg.logRecordProcessors, p...)
		return nil
	})
}

func WithTracerProvider(tp *sdktrace.TracerProvider) Option {
	return optionFunc(func(cfg *config) error {
		cfg.tracerProvider = tp
		return nil
	})
}

func WithMeterProvider(mp *sdkmetric.MeterProvider) Option {
	return optionFunc(func(cfg *config) error {
		cfg.meterProvider = mp
		return nil
	})
}

func WithLogProvider(lp *sdklog.LoggerProvider) Option {
	return optionFunc(func(cfg *config) error {
		cfg.loggerProvider = lp
		return nil
	})
}
