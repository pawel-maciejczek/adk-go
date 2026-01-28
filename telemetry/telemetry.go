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

// Package telemetry allows to set up custom telemetry processors that the ADK events
// will be emitted to.
package telemetry

import (
	"context"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	internal "google.golang.org/adk/internal/telemetry"
)

const (
	SystemName = internal.SystemName
)

// RegisterSpanProcessor registers the span processor to local trace provider instance.
// Any processor should be registered BEFORE any of the events are emitted, otherwise
// the registration will be ignored.
// In addition to the RegisterSpanProcessor function, global trace provider configs
// are respected.
//
// Deprecated
func RegisterSpanProcessor(processor sdktrace.SpanProcessor) {
	internal.AddSpanProcessor(processor)
}

func Configure(ctx context.Context, opts ...Option) (*config, error) {
	return configureInternal(ctx, opts...)
}

// Telemetry wraps all telemetry providers and implements functions for telemetry lifecycle management.
type Telemetry interface {
	// SetGlobalProviders sets the configured providers as global OTel registry.
	SetGlobalProviders()
	// TraceProvider returns the configured TraceProvider or nil.
	TraceProvider() *sdktrace.TracerProvider
	// TraceProvider returns the configured MeterProvider or nil.
	MeterProvider() *sdkmetric.MeterProvider
	// TraceProvider returns the configured LoggerProvider or nil.
	LoggerProvider() *sdklog.LoggerProvider
	// Shutdown shuts down underlying OTel providers.
	Shutdown(ctx context.Context) error
}

// New initializes new telemetry.
// By default it initializes TraceProvider, LogProvider and MeterProvider.
// Options can be used to customize the defaults, e.g. use custom credentials, add SpanProcessors or use preconfigured TraceProvider.
// Telemetry providers should be installed in otel using InstallGlobal() function.
//
// # Usage
//
//	 func main() {
//			telemetry, err := telemetry.New(ctx,
//				telemetry.WithOtelToCloud(true),
//				telemetry.WithResource(resource.NewWithAttributes(
//					semconv.SchemaURL,
//					attribute.String("service.name", "my-service"),
//				)),
//			)
//			if err != nil {
//				log.Fatal(err)
//			}
//			defer telemetry.Shutdown(context.WithoutCancel(ctx))
//			telemetry.InstallGlobal()
//			// app code
//		}
//
// The caller must call Shutdown() method to gracefully shutdown underlying telemetry and release resources.
func New(ctx context.Context, opts ...Option) (Telemetry, error) {
	cfg, err := Configure(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return newInternal(ctx, cfg)
}
