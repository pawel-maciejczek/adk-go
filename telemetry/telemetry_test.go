// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package telemetry

import (
	"context"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestTelemetrySmoke(t *testing.T) {
	exporter := newMockExporter()
	ctx := t.Context()

	// Initialize telemetry.
	telemetry, err := New(t.Context(), WithSpanProcessors(sdktrace.NewSimpleSpanProcessor(exporter)))
	if err != nil {
		t.Fatalf("failed to create telemetry: %v", err)
	}
	t.Cleanup(func() {
		telemetry.Shutdown(context.WithoutCancel(ctx))
	})
	telemetry.SetGlobalOtelProviders()

	// Create test tracer.
	tracer := otel.Tracer("test-tracer")
	spanName := "test-span"

	_, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
	span.End()

	if err := telemetry.TraceProvider().ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush spans: %v", err)
	}

	// Check exporter contains the span.
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if spans[0].Name() != spanName {
		t.Errorf("got span name %q, want %q", spans[0].Name(), spanName)
	}

	telemetry.Shutdown(context.WithoutCancel(ctx))
	if exporter.running == true {
		t.Errorf("Expected test exporter to be not running after shutdown")
	}
}

func newMockExporter() *mockExporter {
	return &mockExporter{
		running: true,
	}
}

// mockExporter is a simple in-memory span exporter for testing.
type mockExporter struct {
	mu      sync.Mutex
	spans   []sdktrace.ReadOnlySpan
	running bool
}

func (e *mockExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return nil
	}
	e.spans = append(e.spans, spans...)
	return nil
}

func (e *mockExporter) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.running = false
	return nil
}

func (e *mockExporter) GetSpans() []sdktrace.ReadOnlySpan {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.spans
}
