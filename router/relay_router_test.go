package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetRelayRouterRegistersAnthropicCountTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	SetRelayRouter(router)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatalf("expected /v1/messages/count_tokens to be registered, got 404")
	}
}
