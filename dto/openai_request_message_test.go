package dto

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestMessagePreservesExplicitEmptyReasoningFields(t *testing.T) {
	payload := []byte(`{"role":"assistant","content":"hello","reasoning_content":"","reasoning":""}`)

	var message Message
	if err := common.Unmarshal(payload, &message); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	encoded, err := common.Marshal(message)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	encodedText := string(encoded)

	if !strings.Contains(encodedText, `"reasoning_content":""`) {
		t.Fatalf("encoded message = %s, want explicit empty reasoning_content", encodedText)
	}
	if !strings.Contains(encodedText, `"reasoning":""`) {
		t.Fatalf("encoded message = %s, want explicit empty reasoning", encodedText)
	}
}

func TestMessageOmitsAbsentReasoningFields(t *testing.T) {
	payload := []byte(`{"role":"assistant","content":"hello"}`)

	var message Message
	if err := common.Unmarshal(payload, &message); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	encoded, err := common.Marshal(message)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	encodedText := string(encoded)

	if strings.Contains(encodedText, "reasoning_content") {
		t.Fatalf("encoded message = %s, want absent reasoning_content omitted", encodedText)
	}
	if strings.Contains(encodedText, "reasoning") {
		t.Fatalf("encoded message = %s, want absent reasoning omitted", encodedText)
	}
}
