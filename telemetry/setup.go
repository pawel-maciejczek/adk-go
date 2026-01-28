package telemetry

// import (
// 	"context"
// 	"os"

// 	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
// 	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
// 	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
// 	"go.opentelemetry.io/otel/sdk/resource"

// 	sdklog "go.opentelemetry.io/otel/sdk/log"
// 	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
// 	sdktrace "go.opentelemetry.io/otel/sdk/trace"
// )

// type Config struct {
// 	spanProcessors      []sdktrace.SpanProcessor
// 	metricReaders       []sdkmetric.Reader
// 	logRecordProcessors []sdklog.Processor
// 	resource            *resource.Resource
// 	otelToCloud         bool
// 	// TODO semconv from env
// }

// type Option interface {
// 	apply(Config) Config
// }

// type optionFunc func(Config) Config

// func (fn optionFunc) apply(cfg Config) Config {
// 	return fn(cfg)
// }

// func WithSpanProcessor(spanProcessor sdktrace.SpanProcessor) Option {
// 	return optionFunc(func(cfg Config) Config {
// 		cfg.spanProcessors = append(cfg.spanProcessors, spanProcessor)
// 		return cfg
// 	})
// }

// func WithMetricReader(metricReader sdkmetric.Reader) Option {
// 	return optionFunc(func(cfg Config) Config {
// 		cfg.metricReaders = append(cfg.metricReaders, metricReader)
// 		return cfg
// 	})
// }

// func WithLogRecordProcessor(logRecordProcessor sdklog.Processor) Option {
// 	return optionFunc(func(cfg Config) Config {
// 		cfg.logRecordProcessors = append(cfg.logRecordProcessors, logRecordProcessor)
// 		return cfg
// 	})
// }

// func WithOtelResource(otelResource *resource.Resource) Option {
// 	return optionFunc(func(cfg Config) Config {
// 		cfg.resource = otelResource
// 		return cfg
// 	})
// }

// // Setup sets up Open Telemetry based on the configured options.
// func Setup(ctx context.Context, opts ...Option) (shutdown func(), err error) {
// 	cfg, err := newConfig(ctx, opts...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var otelShutdown []func(context.Context) error
// 	if len(cfg.spanProcessors) > 0 {
// 		// TODO register
// 	}
// 	if len(cfg.metricReaders) > 0 {
// 		// TODO register
// 	}
// 	if len(cfg.logRecordProcessors) > 0 {
// 		// TODO register
// 	}
// 	return func() {
// 		for _, f := range otelShutdown {
// 			f(context.Background())
// 		}
// 	}, nil
// }

// func newConfig(ctx context.Context, opts ...Option) (*Config, error) {
// 	cfg := Config{}
// 	for _, opt := range opts {
// 		cfg = opt.apply(cfg)
// 	}
// 	if cfg.resource == nil {
// 		// TODO add detectors
// 		resource, err := resource.Detect(ctx)
// 		if err != nil {
// 			return nil, err
// 		}
// 		cfg.resource = resource
// 	}
// 	_, otelEndpointExists := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
// 	_, otelTracesEndpointExists := os.LookupEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
// 	_, otelMetricsEndpointExists := os.LookupEnv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT")
// 	_, otelLogsEndpointExists := os.LookupEnv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT")
// 	// TODO order
// 	if otelEndpointExists || otelTracesEndpointExists {
// 		exporter, err := otlptracehttp.New(ctx)
// 		if err != nil {
// 			return nil, err
// 		}
// 		cfg.spanProcessors = append(cfg.spanProcessors, sdktrace.NewBatchSpanProcessor(
// 			exporter,
// 		))
// 	}
// 	if otelEndpointExists || otelMetricsEndpointExists {
// 		reader, err := otlpmetrichttp.New(ctx)
// 		if err != nil {
// 			return nil, err
// 		}
// 		cfg.metricReaders = append(cfg.metricReaders, sdkmetric.NewPeriodicReader(reader))
// 	}
// 	if otelEndpointExists || otelLogsEndpointExists {
// 		processor, err := otlploghttp.New(ctx)
// 		if err != nil {
// 			return nil, err
// 		}
// 		cfg.logRecordProcessors = append(cfg.logRecordProcessors, sdklog.NewBatchProcessor(processor))
// 	}
// 	// TODO install otel to cloud from internal
// 	return &cfg, nil
// }
