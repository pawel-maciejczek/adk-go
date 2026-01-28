package telemetry

import (
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/oauth2/google"
)

type config struct {
	// Controls whether to configure exports to GCP.
	oTelToCloud bool

	// Credentials override the default credentials.
	Credentials *google.Credentials
	// TODO Resource
	Resource            *resource.Resource
	SpanProcessors      []sdktrace.SpanProcessor
	MetricReaders       []sdkmetric.Reader
	LogRecordProcessors []sdklog.Processor

	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
}

// Option configures a adk telemetry.
type Option interface {
	apply(*config) error
}

type optionFunc func(*config) error

func (fn optionFunc) apply(cfg *config) error {
	return fn(cfg)
}

// WithOtelToCloud enables exporting telemetry to GCP.
func WithOtelToCloud(value bool) Option {
	return optionFunc(func(cfg *config) error {
		cfg.oTelToCloud = value
		return nil
	})
}

// WithResource configures the OTel resource.
func WithResource(r *resource.Resource) Option {
	return optionFunc(func(cfg *config) error {
		cfg.Resource = r
		return nil
	})
}

// WithCredentials allows to pass custom credentials to OTel exporters.
func WithCredentials(c *google.Credentials) Option {
	return optionFunc(func(cfg *config) error {
		cfg.Credentials = c
		return nil
	})
}

// WithSpanProcessors registers additional span processors.
func WithSpanProcessors(p ...sdktrace.SpanProcessor) Option {
	return optionFunc(func(cfg *config) error {
		cfg.SpanProcessors = append(cfg.SpanProcessors, p...)
		return nil
	})
}

// WithMetricReaders registers additional metric readers.
func WithMetricReaders(r ...sdkmetric.Reader) Option {
	return optionFunc(func(cfg *config) error {
		cfg.MetricReaders = append(cfg.MetricReaders, r...)
		return nil
	})
}

// WithLogRecordProcessors registers additional log record processors.
func WithLogRecordProcessors(p ...sdklog.Processor) Option {
	return optionFunc(func(cfg *config) error {
		cfg.LogRecordProcessors = append(cfg.LogRecordProcessors, p...)
		return nil
	})
}

// WithTracerProvider overrides the default TracerProvider with preconfigured instance.
func WithTracerProvider(tp *sdktrace.TracerProvider) Option {
	return optionFunc(func(cfg *config) error {
		cfg.TracerProvider = tp
		return nil
	})
}

// WithMeterProvider overrides the default MeterProvider with preconfigured instance.
func WithMeterProvider(mp *sdkmetric.MeterProvider) Option {
	return optionFunc(func(cfg *config) error {
		cfg.MeterProvider = mp
		return nil
	})
}

// WithLogProvider overrides the detault LoggerProvider with preconfigured instance.
func WithLoggerProvider(lp *sdklog.LoggerProvider) Option {
	return optionFunc(func(cfg *config) error {
		cfg.LoggerProvider = lp
		return nil
	})
}
