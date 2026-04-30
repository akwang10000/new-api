package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestOpenAIResponsesResponseAcceptsNullInstructions(t *testing.T) {
	payload := []byte(`{
		"id":"resp_1",
		"object":"response",
		"created_at":1,
		"status":"completed",
		"instructions":null,
		"max_output_tokens":16,
		"model":"gpt-4.1",
		"output":[],
		"parallel_tool_calls":false,
		"previous_response_id":null,
		"reasoning":null,
		"store":false,
		"temperature":1,
		"tool_choice":"auto",
		"tools":[],
		"top_p":1,
		"truncation":"disabled",
		"usage":null,
		"user":null,
		"metadata":null
	}`)

	var response OpenAIResponsesResponse
	if err := common.Unmarshal(payload, &response); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if string(response.Instructions) != "null" {
		t.Fatalf("Instructions = %s, want null", response.Instructions)
	}
}
