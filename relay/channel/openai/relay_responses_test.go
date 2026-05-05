package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

func TestOaiResponsesStreamHandlerEstimatesUsageWhenUpstreamOmitsUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 300
	t.Cleanup(func() { constant.StreamingTimeout = oldStreamingTimeout })

	body := strings.Join([]string{
		`data: {"type":"response.output_text.delta","delta":"hello world"}`,
		`data: {"type":"response.completed","response":{"id":"resp_1","object":"response"}}`,
		`data: [DONE]`,
		``,
	}, "\n")
	resp := &http.Response{Body: io.NopCloser(strings.NewReader(body))}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "test-model"},
	}
	info.SetEstimatePromptTokens(10)

	usage, apiErr := OaiResponsesStreamHandler(ctx, info, resp)
	if apiErr != nil {
		t.Fatalf("OaiResponsesStreamHandler returned error: %v", apiErr)
	}
	if usage.CompletionTokens <= 0 {
		t.Fatalf("CompletionTokens = %d, want estimated completion tokens", usage.CompletionTokens)
	}
	if usage.PromptTokens != 10 {
		t.Fatalf("PromptTokens = %d, want estimate prompt tokens", usage.PromptTokens)
	}
	if usage.TotalTokens != usage.PromptTokens+usage.CompletionTokens {
		t.Fatalf("TotalTokens = %d, want prompt + completion (%d)", usage.TotalTokens, usage.PromptTokens+usage.CompletionTokens)
	}
}
