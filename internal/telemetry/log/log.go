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

// Package telemetry sets up the open telemetry exporters to the ADK.
//
// WARNING: telemetry provided by ADK (internaltelemetry package) may change (e.g. attributes and their names)
// because we're in process to standardize and unify telemetry across all ADKs.
package log

import (
	"context"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"google.golang.org/adk/internal/telemetry"
)

var logger = otelslog.NewLogger(telemetry.SystemName)

func LoggingBeforeInvokeAgent(ctx context.Context) {
	logger.InfoContext(ctx, "BeforeInvokeAgent")
}

func LoggingAfterInvokeAgent(ctx context.Context) {
	logger.InfoContext(ctx, "AfterInvokeAgent")
}

func LoggingBeforeInvokeModel(ctx context.Context) {
	logger.InfoContext(ctx, "BeforeInvokeModel")
}

func LoggingAfterInvokeModel(ctx context.Context) {
	logger.InfoContext(ctx, "AfterInvokeModel")
}

func LoggingBeforeInvokeTool(ctx context.Context) {
	logger.InfoContext(ctx, "BeforeInvokeTool")
}

func LoggingAfterInvokeTool(ctx context.Context) {
	logger.InfoContext(ctx, "BeforeInvokeTool")
}
