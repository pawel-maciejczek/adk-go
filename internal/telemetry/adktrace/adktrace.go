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
package adktrace

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.36.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/adk/internal/telemetry"
	"google.golang.org/adk/internal/version"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

const (
	systemName           = telemetry.SystemName
	genAiOperationName   = "gen_ai.operation.name"
	genAiToolDescription = "gen_ai.tool.description"
	genAiToolName        = "gen_ai.tool.name"
	genAiToolCallID      = "gen_ai.tool.call.id"
	genAiSystemName      = "gen_ai.system"

	genAiRequestModelName = "gen_ai.request.model"
	genAiRequestTopP      = "gen_ai.request.top_p"
	genAiRequestMaxTokens = "gen_ai.request.max_tokens"

	genAiResponseFinishReason            = "gen_ai.response.finish_reason"
	genAiResponsePromptTokenCount        = "gen_ai.response.prompt_token_count"
	genAiResponseCandidatesTokenCount    = "gen_ai.response.candidates_token_count"
	genAiResponseCachedContentTokenCount = "gen_ai.response.cached_content_token_count"
	genAiResponseTotalTokenCount         = "gen_ai.response.total_token_count"
	genAiConversationID                  = "gen_ai.conversation.id"

	gcpVertexAgentLLMRequestName   = "gcp.vertex.agent.llm_request"
	gcpVertexAgentToolCallArgsName = "gcp.vertex.agent.tool_call_args"
	gcpVertexAgentEventID          = "gcp.vertex.agent.event_id"
	gcpVertexAgentToolResponseName = "gcp.vertex.agent.tool_response"
	gcpVertexAgentLLMResponseName  = "gcp.vertex.agent.llm_response"
	gcpVertexAgentInvocationID     = "gcp.vertex.agent.invocation_id"
	gcpVertexAgentSessionID        = "gcp.vertex.agent.session_id"

	executeToolName = "execute_tool"
	mergeToolName   = "(merged tools)"
)

// tracer is the tracer instance for ADK go.
// The default value uses global tracer. adk-go version needs to be initialized in Init function. TODO update docs
var tracer trace.Tracer = otel.GetTracerProvider().Tracer(systemName, trace.WithInstrumentationVersion(version.Version), trace.WithSchemaURL(""))

// StartTrace returns two spans to start emitting events, one from global tracer and second from the local.
func StartTrace(ctx context.Context, traceName string) (context.Context, trace.Span) {
	return tracer.Start(ctx, traceName)
}

// TraceMergedToolCalls traces the tool execution events.
func TraceMergedToolCalls(span trace.Span, fnResponseEvent *session.Event) {
	if fnResponseEvent == nil {
		return
	}
	attributes := []attribute.KeyValue{
		attribute.String(genAiOperationName, executeToolName),
		attribute.String(genAiToolName, mergeToolName),
		attribute.String(genAiToolDescription, mergeToolName),
		// Setting empty llm request and response (as UI expect these) while not
		// applicable for tool_response.
		attribute.String(gcpVertexAgentLLMRequestName, "{}"),
		attribute.String(gcpVertexAgentLLMRequestName, "{}"),
		attribute.String(gcpVertexAgentToolCallArgsName, "N/A"),
		attribute.String(gcpVertexAgentEventID, fnResponseEvent.ID),
		attribute.String(gcpVertexAgentToolResponseName, safeSerialize(fnResponseEvent)),
	}
	span.SetAttributes(attributes...)
	span.End()
}

type traceableTool interface {
	Name() string
	Description() string
}

// TraceToolCall traces the tool execution events.
func TraceToolCall(span trace.Span, tool traceableTool, fnArgs map[string]any, fnResponseEvent *session.Event) {
	if fnResponseEvent == nil {
		return
	}
	attributes := []attribute.KeyValue{
		attribute.String(genAiOperationName, executeToolName),
		attribute.String(genAiToolName, tool.Name()),
		attribute.String(genAiToolDescription, tool.Description()),
		// TODO: add tool type

		// Setting empty llm request and response (as UI expect these) while not
		// applicable for tool_response.
		attribute.String(gcpVertexAgentLLMRequestName, "{}"),
		attribute.String(gcpVertexAgentLLMRequestName, "{}"),
		attribute.String(gcpVertexAgentToolCallArgsName, safeSerialize(fnArgs)),
		attribute.String(gcpVertexAgentEventID, fnResponseEvent.ID),
	}

	toolCallID := "<not specified>"
	toolResponse := "<not specified>"

	if fnResponseEvent.LLMResponse.Content != nil {
		responseParts := fnResponseEvent.LLMResponse.Content.Parts

		if len(responseParts) > 0 {
			functionResponse := responseParts[0].FunctionResponse
			if functionResponse != nil {
				if functionResponse.ID != "" {
					toolCallID = functionResponse.ID
				}
				if functionResponse.Response != nil {
					toolResponse = safeSerialize(functionResponse.Response)
				}
			}
		}
	}

	attributes = append(attributes, attribute.String(genAiToolCallID, toolCallID))
	attributes = append(attributes, attribute.String(gcpVertexAgentToolResponseName, toolResponse))

	span.SetAttributes(attributes...)
	span.End()
}

// TraceLLMCall fills the call_llm event details.
func TraceLLMCall(span trace.Span, sessionID string, llmRequest *model.LLMRequest, event *session.Event) {
	attributes := []attribute.KeyValue{
		attribute.String(genAiSystemName, telemetry.SystemName),
		attribute.String(genAiRequestModelName, llmRequest.Model),
		attribute.String(gcpVertexAgentInvocationID, event.InvocationID),
		attribute.String(gcpVertexAgentSessionID, sessionID),
		attribute.String(genAiConversationID, sessionID),
		attribute.String(gcpVertexAgentEventID, event.ID),
		attribute.String(gcpVertexAgentLLMRequestName, safeSerialize(llmRequestToTrace(llmRequest))),
		attribute.String(gcpVertexAgentLLMResponseName, safeSerialize(event.LLMResponse)),
	}

	if llmRequest.Config.TopP != nil {
		attributes = append(attributes, attribute.Float64(genAiRequestTopP, float64(*llmRequest.Config.TopP)))
	}

	if llmRequest.Config.MaxOutputTokens != 0 {
		attributes = append(attributes, attribute.Int(genAiRequestMaxTokens, int(llmRequest.Config.MaxOutputTokens)))
	}
	if event.FinishReason != "" {
		attributes = append(attributes, attribute.String(genAiResponseFinishReason, string(event.FinishReason)))
	}
	if event.UsageMetadata != nil {
		if event.UsageMetadata.PromptTokenCount > 0 {
			attributes = append(attributes, attribute.Int(genAiResponsePromptTokenCount, int(event.UsageMetadata.PromptTokenCount)))
		}
		if event.UsageMetadata.CandidatesTokenCount > 0 {
			attributes = append(attributes, attribute.Int(genAiResponseCandidatesTokenCount, int(event.UsageMetadata.CandidatesTokenCount)))
		}
		if event.UsageMetadata.CachedContentTokenCount > 0 {
			attributes = append(attributes, attribute.Int(genAiResponseCachedContentTokenCount, int(event.UsageMetadata.CachedContentTokenCount)))
		}
		if event.UsageMetadata.TotalTokenCount > 0 {
			attributes = append(attributes, attribute.Int(genAiResponseTotalTokenCount, int(event.UsageMetadata.TotalTokenCount)))
		}
	}

	span.SetAttributes(attributes...)
	span.End()
}

func safeSerialize(obj any) string {
	dump, err := json.Marshal(obj)
	if err != nil {
		return "<not serializable>"
	}
	return string(dump)
}

func llmRequestToTrace(llmRequest *model.LLMRequest) map[string]any {
	result := map[string]any{
		"config":  llmRequest.Config,
		"model":   llmRequest.Model,
		"content": []*genai.Content{},
	}
	for _, content := range llmRequest.Contents {
		parts := []*genai.Part{}
		// filter out InlineData part
		for _, part := range content.Parts {
			if part.InlineData != nil {
				continue
			}
			parts = append(parts, part)
		}
		filteredContent := &genai.Content{
			Role:  content.Role,
			Parts: parts,
		}
		result["content"] = append(result["content"].([]*genai.Content), filteredContent)
	}
	return result
}

type InvokeAgentParams struct {
	AgentName        string
	AgentDescription string
	SessionID        string
}

// StartInvokeAgent starts the invoke_agent span.
func StartInvokeAgent(ctx context.Context, params InvokeAgentParams) (context.Context, trace.Span) {
	agentName := params.AgentName
	spanCtx, span := tracer.Start(ctx, fmt.Sprintf("invoke_agent %s", agentName), trace.WithAttributes(
		semconv.GenAIOperationNameInvokeAgent,
		semconv.GenAIAgentDescription(params.AgentDescription),
		semconv.GenAIAgentName(agentName),
		semconv.GenAIConversationID(params.SessionID),
	))

	return spanCtx, span
}

func AfterInvokeAgent(span trace.Span, e *session.Event, err error) {
	recordErrorAndStatus(span, err)
	// TODO responses, etc
}

type GenerateContentParams struct {
	ModelName string
}

// StartInvokeModel starts new generate_content span.
func StartGenerateContent(ctx context.Context, params GenerateContentParams) (context.Context, trace.Span) {
	modelName := params.ModelName
	spanCtx, span := tracer.Start(ctx, fmt.Sprintf("generate_content %s", modelName), trace.WithAttributes(
		semconv.GenAIOperationNameGenerateContent,
		semconv.GenAIRequestModel(modelName),
		semconv.GenAIUsageInputTokens(123),
	))
	return spanCtx, span
}

func AfterGenerateContent(span trace.Span, resp *model.LLMResponse, err error) {
	recordErrorAndStatus(span, err)
	span.SetAttributes(
		semconv.GenAIUsageOutputTokens(123),
		semconv.GenAIResponseFinishReasons(""),
		attribute.String("gcp.vertex.agent.event_id", "TODO"),
	)
}

type ExecuteToolParams struct {
	ToolName  string
	ModelName string
}

// StartExecuteTool starts new execute_tool span.
func StartExecuteTool(ctx context.Context, params ExecuteToolParams) (context.Context, trace.Span) {
	toolName := params.ToolName
	modelName := params.ModelName
	spanCtx, span := tracer.Start(ctx, fmt.Sprintf("execute_tool %s", toolName), trace.WithAttributes(
		semconv.GenAIOperationNameExecuteTool,
		semconv.GenAIRequestModel(modelName),
		semconv.GenAIUsageInputTokens(123),
	))
	return spanCtx, span
}

func AfterExecuteTool(span trace.Span, result map[string]any) {
	var err error
	errVal, _ := result["error"]
	if errVal != nil {
		err = errVal.(error)
	}
	recordErrorAndStatus(span, err)
}

// TraceMergedToolCalls traces the tool execution events.
func StartMergedToolCalls(ctx context.Context) (context.Context, trace.Span) {
	mergedToolName := "(merged tools)"
	ctx, span := tracer.Start(ctx, "execute_tool (merged)", trace.WithAttributes(
		semconv.GenAIOperationNameExecuteTool,
		semconv.GenAIToolName(mergedToolName),
		semconv.GenAIToolDescription(mergedToolName),
		// Setting empty llm request and response (as UI expect these) while not
		// applicable for tool_response.
		attribute.String(gcpVertexAgentLLMRequestName, "{}"),
		attribute.String(gcpVertexAgentLLMRequestName, "{}"),
		attribute.String(gcpVertexAgentToolCallArgsName, "N/A"),
	))
	return ctx, span
}

func AfterMergedToolCalls(span trace.Span, fnResponseEvent *session.Event, err error) {
	if fnResponseEvent == nil {
		return
	}
	recordErrorAndStatus(span, err)
	span.SetAttributes(
		attribute.String(gcpVertexAgentEventID, fnResponseEvent.ID),
		attribute.String(gcpVertexAgentToolResponseName, safeSerialize(fnResponseEvent)),
	)
}

func recordErrorAndStatus(span trace.Span, err error) {
	span.RecordError(err)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
}
