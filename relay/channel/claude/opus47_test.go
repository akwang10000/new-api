package claude

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func TestRequestOpenAI2ClaudeMessageOpus47ThinkingUsesAdaptiveAndOmitsSampling(t *testing.T) {
	temperature := 0.2
	topP := 0.3
	topK := 7
	maxTokens := uint(4096)

	claudeRequest, err := RequestOpenAI2ClaudeMessage(nil, dto.GeneralOpenAIRequest{
		Model:       "claude-opus-4-7-thinking",
		Messages:    []dto.Message{{Role: "user", Content: "hello"}},
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
		TopP:        &topP,
		TopK:        &topK,
	})
	if err != nil {
		t.Fatalf("RequestOpenAI2ClaudeMessage returned error: %v", err)
	}
	if claudeRequest.Model != "claude-opus-4-7" {
		t.Fatalf("Model = %q, want claude-opus-4-7", claudeRequest.Model)
	}
	if claudeRequest.Thinking == nil {
		t.Fatal("Thinking is nil")
	}
	if claudeRequest.Thinking.Type != "adaptive" {
		t.Fatalf("Thinking.Type = %q, want adaptive", claudeRequest.Thinking.Type)
	}
	if claudeRequest.Thinking.Display != "summarized" {
		t.Fatalf("Thinking.Display = %q, want summarized", claudeRequest.Thinking.Display)
	}
	if claudeRequest.Temperature != nil {
		t.Fatalf("Temperature = %v, want nil", *claudeRequest.Temperature)
	}
	if claudeRequest.TopP != nil {
		t.Fatalf("TopP = %v, want nil", *claudeRequest.TopP)
	}
	if claudeRequest.TopK != nil {
		t.Fatalf("TopK = %v, want nil", *claudeRequest.TopK)
	}

	var outputConfig struct {
		Effort string `json:"effort"`
	}
	if err := common.Unmarshal(claudeRequest.OutputConfig, &outputConfig); err != nil {
		t.Fatalf("failed to unmarshal output_config: %v", err)
	}
	if outputConfig.Effort != "high" {
		t.Fatalf("output_config.effort = %q, want high", outputConfig.Effort)
	}
}

func TestRequestOpenAI2ClaudeMessageOpus47XHighSuffixUsesAdaptive(t *testing.T) {
	claudeRequest, err := RequestOpenAI2ClaudeMessage(nil, dto.GeneralOpenAIRequest{
		Model:    "claude-opus-4-7-xhigh",
		Messages: []dto.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("RequestOpenAI2ClaudeMessage returned error: %v", err)
	}
	if claudeRequest.Model != "claude-opus-4-7" {
		t.Fatalf("Model = %q, want claude-opus-4-7", claudeRequest.Model)
	}
	if claudeRequest.Thinking == nil || claudeRequest.Thinking.Type != "adaptive" || claudeRequest.Thinking.Display != "summarized" {
		t.Fatalf("Thinking = %#v, want adaptive summarized", claudeRequest.Thinking)
	}

	var outputConfig struct {
		Effort string `json:"effort"`
	}
	if err := common.Unmarshal(claudeRequest.OutputConfig, &outputConfig); err != nil {
		t.Fatalf("failed to unmarshal output_config: %v", err)
	}
	if outputConfig.Effort != "xhigh" {
		t.Fatalf("output_config.effort = %q, want xhigh", outputConfig.Effort)
	}
}
