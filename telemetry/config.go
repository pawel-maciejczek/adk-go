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
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/oauth2/google"
)

type config struct {
	// Enables/disables telemetry export to GCP.
	oTelToCloud bool

	// TODO should it be GOOGLE_CLOUD_PROJECT env var, which is used for billing?
	// resourceProjectID is used as the gcp.project.id resource attribute.
	// If it's empty, the project will be read from GOOGLE_CLOUD_PROJECT env variable or ADC.
	resourceProjectID string

	// credentials override the application default credentials.
	credentials *google.Credentials

	// resource allows to customize OTel resource. It will be merged with default detectors.
	resource *resource.Resource
	// spanProcessors allow to register additional span processors, e.g. for custom span exporters.
	spanProcessors []sdktrace.SpanProcessor
	// metricReaders allow to register additional metric readers, e.g. for custom metric exporters.
	metricReaders []sdkmetric.Reader
	// logRecordProcessors allow to register additional log record processors, e.g. for custom log exporters.
	logRecordProcessors []sdklog.Processor

	// tracerProvider overrides the default TracerProvider.
	tracerProvider *sdktrace.TracerProvider
	// meterProvider overrides the default MeterProvider.
	meterProvider *sdkmetric.MeterProvider
	// loggerProvider overrides the default LoggerProvider.
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

// WithOtelToCloud enables exporting telemetry to GCP.
func WithOtelToCloud(value bool) Option {
	return optionFunc(func(cfg *config) error {
		cfg.oTelToCloud = value
		return nil
	})
}

// WithProjectID sets the gcp.project.id resource attribute.
func WithResourceProjectID(projectID string) Option {
	return optionFunc(func(cfg *config) error {
		cfg.resourceProjectID = projectID
		return nil
	})
}

// WithResource configures the OTel resource.
func WithResource(r *resource.Resource) Option {
	return optionFunc(func(cfg *config) error {
		cfg.resource = r
		return nil
	})
}

// WithCredentials allows to pass custom credentials to OTel exporters.
func WithCredentials(c *google.Credentials) Option {
	return optionFunc(func(cfg *config) error {
		cfg.credentials = c
		return nil
	})
}

// WithSpanProcessors registers additional span processors.
func WithSpanProcessors(p ...sdktrace.SpanProcessor) Option {
	return optionFunc(func(cfg *config) error {
		cfg.spanProcessors = append(cfg.spanProcessors, p...)
		return nil
	})
}

// WithMetricReaders registers additional metric readers.
func WithMetricReaders(r ...sdkmetric.Reader) Option {
	return optionFunc(func(cfg *config) error {
		cfg.metricReaders = append(cfg.metricReaders, r...)
		return nil
	})
}

// WithLogRecordProcessors registers additional log record processors.
func WithLogRecordProcessors(p ...sdklog.Processor) Option {
	return optionFunc(func(cfg *config) error {
		cfg.logRecordProcessors = append(cfg.logRecordProcessors, p...)
		return nil
	})
}

// WithTracerProvider overrides the default TracerProvider with preconfigured instance.
func WithTracerProvider(tp *sdktrace.TracerProvider) Option {
	return optionFunc(func(cfg *config) error {
		cfg.tracerProvider = tp
		return nil
	})
}

// WithMeterProvider overrides the default MeterProvider with preconfigured instance.
func WithMeterProvider(mp *sdkmetric.MeterProvider) Option {
	return optionFunc(func(cfg *config) error {
		cfg.meterProvider = mp
		return nil
	})
}

// WithLogProvider overrides the detault LoggerProvider with preconfigured instance.
func WithLoggerProvider(lp *sdklog.LoggerProvider) Option {
	return optionFunc(func(cfg *config) error {
		cfg.loggerProvider = lp
		return nil
	})
}
