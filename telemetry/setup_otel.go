// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package telemetry implements the open telemetry in ADK.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.36.0"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func newInternal(ctx context.Context, cfg *config) (*telemetryService, error) {
	tp, err := initTracerProvider(ctx, cfg)
	if err != nil {
		return nil, err
	}
	// TODO init logger provider
	// TODO init meter provider

	return &telemetryService{
		tp: tp,
		mp: cfg.meterProvider,
		lp: cfg.loggerProvider,
	}, nil
}

type telemetryService struct {
	tp *sdktrace.TracerProvider
	mp *sdkmetric.MeterProvider
	lp *sdklog.LoggerProvider
}

func (t *telemetryService) TraceProvider() *sdktrace.TracerProvider {
	return t.tp
}

func (t *telemetryService) MeterProvider() *sdkmetric.MeterProvider {
	return t.mp
}

func (t *telemetryService) LoggerProvider() *sdklog.LoggerProvider {
	return t.lp
}

func (t *telemetryService) Shutdown(ctx context.Context) error {
	var err error
	if t.tp != nil {
		errors.Join(err, t.tp.Shutdown(ctx))
	}
	if t.mp != nil {
		errors.Join(err, t.mp.Shutdown(ctx))
	}
	if t.lp != nil {
		errors.Join(err, t.lp.Shutdown(ctx))
	}
	return err
}

func (t *telemetryService) SetGlobalOtelProviders() {
	if t.tp != nil {
		otel.SetTracerProvider(t.tp)
	}
	if t.mp != nil {
		otel.SetMeterProvider(t.mp)
	}
	if t.lp != nil {
		global.SetLoggerProvider(t.lp)
	}
}

func configure(ctx context.Context, opts ...Option) (*config, error) {
	cfg := &config{}
	optsFromEnv, err := otelProvidersFromEnv(ctx)
	if err != nil {
		return &config{}, err
	}
	opts = append(opts, optsFromEnv...)

	for _, opt := range opts {
		if err := opt.apply(cfg); err != nil {
			return &config{}, err
		}
	}

	if cfg.oTelToCloud {
		// Load ADC if no credentials were provided in config.
		if cfg.credentials == nil {
			cfg.credentials, err = loadApplicationDefaultCredentials(ctx)
			if err != nil {
				return &config{}, err
			}
		}
	}
	cfg.resource, err = resolveResource(ctx, cfg)
	if err != nil {
		return &config{}, err
	}
	return cfg, nil
}

func resolveResource(ctx context.Context, cfg *config) (*resource.Resource, error) {
	res, err := resource.New(
		ctx,
		resource.WithSchemaURL(semconv.SchemaURL), // TODO fix conflict
		resource.WithFromEnv(),                    // Discover and provide attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables.
		resource.WithTelemetrySDK(),               // Discover and provide information about the OpenTelemetry SDK used.
		resource.WithProcess(),                    // Discover and provide process information.
		resource.WithOS(),                         // Discover and provide OS information.
		resource.WithContainer(),                  // Discover and provide container information.
		resource.WithHost(),                       // Discover and provide host information.
		resource.WithDetectors(gcp.NewDetector(), newProjectIDDetector(cfg)), // Add GCP specific detectors.
	)
	if errors.Is(err, resource.ErrPartialResource) || errors.Is(err, resource.ErrSchemaURLConflict) {
		log.Println(err) // Log non-fatal issues.
	} else if err != nil {
		return nil, err
	}
	return res, nil
}

// otelProvidersFromEnv initializes OTel exporters from environment variables
func otelProvidersFromEnv(ctx context.Context) ([]Option, error) {
	var opts []Option

	_, otelEndpointExists := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_, otelTracesEndpointExists := os.LookupEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	if otelEndpointExists || otelTracesEndpointExists {
		exporter, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, err
		}
		opts = append(opts, WithSpanProcessors(sdktrace.NewBatchSpanProcessor(
			exporter,
		)))

	}
	// TODO initialize meter and logger providers.
	return opts, nil
}

func loadApplicationDefaultCredentials(ctx context.Context) (*google.Credentials, error) {
	adc, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, err
	}
	return adc, nil
}

func initTracerProvider(ctx context.Context, cfg *config) (*sdktrace.TracerProvider, error) {
	if len(cfg.spanProcessors) == 0 {
		return nil, nil
	}
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(cfg.resource),
	}
	for _, p := range cfg.spanProcessors {
		opts = append(opts, sdktrace.WithSpanProcessor(p))
	}

	if cfg.oTelToCloud {
		exporter, err := initGcpSpanExporter(ctx, cfg.credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCP span exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	tp := sdktrace.NewTracerProvider(
		opts...,
	)

	return tp, nil
}

func initGcpSpanExporter(ctx context.Context, credentials *google.Credentials) (sdktrace.SpanExporter, error) {
	// The quota poject is not present in credentials, even when it's set in ADC JSON file.
	// Use env variable to get the quota project as a workaround.
	// Note - the quota project can be different that the OTel resource project, so we don't use the value from the config here.
	projectID, ok := os.LookupEnv("GOOGLE_CLOUD_PROJECT")
	if !ok {
		return nil, fmt.Errorf("telemetry.googleapis.com requires setting the quota project. Please set GOOGLE_CLOUD_PROJECT environment variable.")
	}
	client := oauth2.NewClient(ctx, credentials.TokenSource)
	return otlptracehttp.New(ctx,
		otlptracehttp.WithHTTPClient(client),
		otlptracehttp.WithEndpointURL("https://telemetry.googleapis.com/v1/traces"),
		// Pass the quota project id in headers to fix auth errors.
		// https://cloud.google.com/docs/authentication/adc-troubleshooting/user-creds
		otlptracehttp.WithHeaders(map[string]string{
			"x-goog-user-project": projectID,
		}))
}
