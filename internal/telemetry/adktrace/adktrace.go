package adktrace

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.36.0"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/codes"
	"google.golang.org/adk/internal/version"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/adk/telemetry"
)

const (
	SystemName           = "gcp.vertex.agent"
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
var tracer trace.Tracer = otel.GetTracerProvider().Tracer(telemetry.SystemName, trace.WithInstrumentationVersion(version.Version), trace.WithSchemaURL(""))

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

func safeSerialize(obj any) string {
	dump, err := json.Marshal(obj)
	if err != nil {
		return "<not serializable>"
	}
	return string(dump)
}
