package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func resetRegisterRateLimiterForTest(t *testing.T) {
	t.Helper()
	originalRedisEnabled := common.RedisEnabled
	inMemoryRateLimiter = common.InMemoryRateLimiter{}
	common.RedisEnabled = false
	t.Cleanup(func() {
		inMemoryRateLimiter = common.InMemoryRateLimiter{}
		common.RedisEnabled = originalRedisEnabled
	})
}

func performRegisterRateLimitRequest(router http.Handler, username string) int {
	body := fmt.Sprintf(`{"username":"%s","email":"%s@example.com"}`, username, username)
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.10:12345"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder.Code
}

func TestRegisterRateLimitLimitsRotatingUsernamesByIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetRegisterRateLimiterForTest(t)

	router := gin.New()
	router.POST("/register", RegisterRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 0; i < RegisterIPAttemptMaxRequests; i++ {
		if status := performRegisterRateLimitRequest(router, fmt.Sprintf("user%d", i)); status != http.StatusOK {
			t.Fatalf("request %d: expected status %d, got %d", i+1, http.StatusOK, status)
		}
	}

	if status := performRegisterRateLimitRequest(router, "user-over-limit"); status != http.StatusTooManyRequests {
		t.Fatalf("expected status %d after IP limit, got %d", http.StatusTooManyRequests, status)
	}
}

func TestRegisterCreateRateLimitLimitsNewAccountsByIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetRegisterRateLimiterForTest(t)

	router := gin.New()
	router.POST("/create", func(c *gin.Context) {
		if !CheckRegisterCreateRateLimit(c) {
			return
		}
		c.Status(http.StatusOK)
	})

	for i := 0; i < RegisterIPCreateMaxRequests; i++ {
		req := httptest.NewRequest(http.MethodPost, "/create", nil)
		req.RemoteAddr = "192.0.2.20:12345"
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("create %d: expected status %d, got %d", i+1, http.StatusOK, recorder.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	req.RemoteAddr = "192.0.2.20:12345"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d after create IP limit, got %d", http.StatusTooManyRequests, recorder.Code)
	}
}
