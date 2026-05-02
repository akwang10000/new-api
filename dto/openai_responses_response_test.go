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

func TestResponsesOutputArgumentsStringPreservesObjectArguments(t *testing.T) {
	payload := []byte(`{"type":"function_call","name":"lookup","arguments":{"city":"Paris","days":2}}`)

	var output ResponsesOutput
	if err := common.Unmarshal(payload, &output); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	got := output.ArgumentsString()
	want := `{"city":"Paris","days":2}`
	if got != want {
		t.Fatalf("ArgumentsString() = %s, want %s", got, want)
	}
}

func TestResponsesOutputArgumentsStringDecodesStringArguments(t *testing.T) {
	payload := []byte(`{"type":"function_call","name":"lookup","arguments":"{\"city\":\"Paris\"}"}`)

	var output ResponsesOutput
	if err := common.Unmarshal(payload, &output); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	got := output.ArgumentsString()
	want := `{"city":"Paris"}`
	if got != want {
		t.Fatalf("ArgumentsString() = %s, want %s", got, want)
	}
}

func TestResponsesOutputArgumentsStringReturnsEmptyForNullArguments(t *testing.T) {
	payload := []byte(`{"type":"function_call","name":"lookup","arguments":null}`)

	var output ResponsesOutput
	if err := common.Unmarshal(payload, &output); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if got := output.ArgumentsString(); got != "" {
		t.Fatalf("ArgumentsString() = %q, want empty", got)
	}
}
