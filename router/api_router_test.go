package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetApiRouterDisablesWeChatOAuthRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	SetApiRouter(router)

	testCases := []struct {
		name   string
		method string
		target string
	}{
		{name: "wechat oauth route returns 404", method: http.MethodGet, target: "/api/oauth/wechat"},
		{name: "wechat bind route returns 404", method: http.MethodGet, target: "/api/oauth/wechat/bind"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, tc.target, nil)

			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusNotFound {
				t.Fatalf("expected status %d for %s %s, got %d", http.StatusNotFound, tc.method, tc.target, recorder.Code)
			}
		})
	}
}
