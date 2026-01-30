// Package telemetry contains helper functions for initializing telemetry in other launchers.
package telemetry

import (
	"context"

	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/telemetry"
)

// InitTelemetry contains the shared logic for initializing telemetry.
func InitTelemetry(ctx context.Context, config *launcher.Config, otelToCloud bool) (telemetry.Telemetry, error) {
	if otelToCloud {
		config.TelemetryOptions = append(config.TelemetryOptions, telemetry.WithOtelToCloud(true))
	}
	telemetry, err := telemetry.New(ctx, config.TelemetryOptions...)
	if err != nil {
		return nil, err
	}
	telemetry.SetGlobalOtelProviders()
	return telemetry, nil
}
