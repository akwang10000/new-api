package controller

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestFastTokenCountMetaForPricingResponsesIncludesPromptText(t *testing.T) {
	maxOutputTokens := uint(256)
	request := &dto.OpenAIResponsesRequest{
		Input:           json.RawMessage(`"hello"`),
		Instructions:    json.RawMessage(`"system"`),
		MaxOutputTokens: &maxOutputTokens,
	}

	meta := fastTokenCountMetaForPricing(request)

	if !strings.Contains(meta.CombineText, "hello") {
		t.Fatalf("CombineText = %q, want input text included", meta.CombineText)
	}
	if !strings.Contains(meta.CombineText, "system") {
		t.Fatalf("CombineText = %q, want instructions included", meta.CombineText)
	}
	if meta.MaxTokens != int(maxOutputTokens) {
		t.Fatalf("MaxTokens = %d, want %d", meta.MaxTokens, maxOutputTokens)
	}
}
