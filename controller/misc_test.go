package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

type statusAPIResponse struct {
	Success bool           `json:"success"`
	Data    map[string]any `json:"data"`
}

func TestGetStatusOmitsWeChatFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	t.Cleanup(func() {
		common.OptionMap = originalOptionMap
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)

	GetStatus(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response statusAPIResponse
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("expected success response, got body %s", recorder.Body.String())
	}
	if _, exists := response.Data["wechat_login"]; exists {
		t.Fatalf("expected wechat_login to be omitted, got %v", response.Data["wechat_login"])
	}
	if _, exists := response.Data["wechat_qrcode"]; exists {
		t.Fatalf("expected wechat_qrcode to be omitted, got %v", response.Data["wechat_qrcode"])
	}
}

func TestGetStatusIncludesEnabledChatwoot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalOptionMap := common.OptionMap
	common.OptionMap = map[string]string{
		"ChatwootEnabled":      "true",
		"ChatwootBaseURL":      "https://chatwoot.example.com/",
		"ChatwootWebsiteToken": "test-token",
	}
	t.Cleanup(func() {
		common.OptionMap = originalOptionMap
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)

	GetStatus(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response statusAPIResponse
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	chatwoot, ok := response.Data["chatwoot"].(map[string]any)
	if !ok {
		t.Fatalf("expected chatwoot config in response, got %#v", response.Data["chatwoot"])
	}
	if chatwoot["base_url"] != "https://chatwoot.example.com/" {
		t.Fatalf("unexpected chatwoot base_url: %#v", chatwoot["base_url"])
	}
	if chatwoot["website_token"] != "test-token" {
		t.Fatalf("unexpected chatwoot website_token: %#v", chatwoot["website_token"])
	}
}
