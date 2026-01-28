// Copyright 2025 Google LLC
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
//
// WARNING: telemetry provided by ADK (internaltelemetry package) may change (e.g. attributes and their names)
// because we're in process to standardize and unify telemetry across all ADKs.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func newInternal(ctx context.Context, opts ...Option) (*telemetryService, error) {
	cfg, err := newConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	tp, err := initTracerProvider(ctx, cfg)
	if err != nil {
		return nil, err
	}
	mp, err := initMeterProvider(ctx, cfg)
	if err != nil {
		return nil, err
	}
	lp, err := initLogProvider(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &telemetryService{
		tp: tp,
		mp: mp,
		lp: lp,
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

func (t *telemetryService) InstallGlobal() {
	// Initialize and store global providers.
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

func newConfig(ctx context.Context, opts ...Option) (config, error) {
	cfg := config{}
	for _, opt := range opts {
		if err := opt.apply(&cfg); err != nil {
			return config{}, err
		}
	}
	if cfg.otelToCloud {
		if cfg.credentials == nil {
			adc, err := defaultCredentials(ctx)
			if err != nil {
				return config{}, err
			}
			cfg.credentials = adc
		}
	}
	// TODO resource
	return cfg, nil
}

func defaultCredentials(ctx context.Context) (*google.Credentials, error) {
	// adc, err := credentials.DetectDefault(&credentials.DetectOptions{
	// 	Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
	// })
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to find default credentials: %w", err)
	// }
	adc, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return nil, err
	}

	// TODO replace with ADC - figure out why it's not populated by go auth2 lib
	// projectID, err = adc.ProjectID() != ""
	var ok bool
	adc.ProjectID, ok = os.LookupEnv("GOOGLE_CLOUD_PROJECT")
	if !ok {
		return nil, fmt.Errorf("adk telemetry requires a quota project, which is not set by default. To learn how to set your quota project, see https://cloud.google.com/docs/authentication/adc-troubleshooting/user-creds: %w")
	}
	return adc, nil
}

// ref https://source.corp.google.com/piper///depot/google3/third_party/py/google/adk/telemetry/setup.py;l=157
func initTracerProvider(ctx context.Context, cfg config) (*sdktrace.TracerProvider, error) {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(cfg.resource),
	}
	for _, p := range cfg.spanProcessors {
		opts = append(opts, sdktrace.WithSpanProcessor(p))
	}

	if cfg.otelToCloud {
		projectID := cfg.credentials.ProjectID
		// set OTEL_RESOURCE_ATTRIBUTES="gcp.project_id=<project_id>"
		// set endpoint with OTEL_EXPORTER_OTLP_ENDPOINT=https://<endpoint>
		exporter, err := otlptracehttp.New(ctx,
			otlptracehttp.WithHTTPClient(oauth2.NewClient(ctx, cfg.credentials.TokenSource)),
			otlptracehttp.WithHeaders(map[string]string{
				// TODO do we need it?
				"x-goog-user-project": projectID,
			}))
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	tp := sdktrace.NewTracerProvider(
		opts...,
	)

	return tp, nil
}

func initMeterProvider(ctx context.Context, cfg config) (*sdkmetric.MeterProvider, error) {
	var opts []sdkmetric.Option
	if cfg.otelToCloud {
		// Configure Metric Export to send metrics as OTLP
		reader, err := autoexport.NewMetricReader(ctx)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdkmetric.WithReader(reader))
	}
	mp := sdkmetric.NewMeterProvider(
		opts...,
	)

	return mp, nil
}

func initLogProvider(ctx context.Context, cfg config) (*sdklog.LoggerProvider, error) {
	var opts []sdklog.LoggerProviderOption
	if cfg.otelToCloud {
		projectID := cfg.credentials.ProjectID
		log.Printf("Using project %q for logging.", projectID)
		cloudExporter, err := otlploghttp.New(ctx,
			otlploghttp.WithHTTPClient(oauth2.NewClient(ctx, cfg.credentials.TokenSource)),
			otlploghttp.WithHeaders(map[string]string{
				"x-goog-user-project": projectID,
			}))
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdklog.WithProcessor(sdklog.NewBatchProcessor(cloudExporter)))
	}
	loggerProvider := sdklog.NewLoggerProvider(
		opts...,
	)
	return loggerProvider, nil
}
