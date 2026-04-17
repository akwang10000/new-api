package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	routerpkg "github.com/QuantumNous/new-api/router"
	"github.com/gin-gonic/gin"
)

func assertWebSecurityHeaders(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	testCases := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
	}

	for header, want := range testCases {
		if got := recorder.Header().Get(header); got != want {
			t.Fatalf("expected %s=%q, got %q", header, want, got)
		}
	}
}

func TestSetWebRouterNoRouteBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name         string
		method       string
		target       string
		accept       string
		wantStatus   int
		wantSPA      bool
		wantJSONBody bool
	}{
		{name: "root path serves spa index", method: http.MethodGet, target: "/", accept: "text/html", wantStatus: http.StatusOK, wantSPA: true},
		{name: "frontend route with query serves spa index", method: http.MethodGet, target: "/login?from=home", accept: "text/html", wantStatus: http.StatusOK, wantSPA: true},
		{name: "nested frontend route serves spa index", method: http.MethodGet, target: "/users/123/profile", accept: "text/html", wantStatus: http.StatusOK, wantSPA: true},
		{name: "console route serves spa index", method: http.MethodGet, target: "/console", accept: "text/html", wantStatus: http.StatusOK, wantSPA: true},
		{name: "pricing route serves spa index", method: http.MethodGet, target: "/pricing", accept: "text/html", wantStatus: http.StatusOK, wantSPA: true},
		{name: "register route serves spa index", method: http.MethodGet, target: "/register", accept: "text/html", wantStatus: http.StatusOK, wantSPA: true},
		{name: "register route with aff query returns not found", method: http.MethodGet, target: "/register?aff=hZMJ", accept: "text/html", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "login route with aff query returns not found", method: http.MethodGet, target: "/login?aff=hZMJ", accept: "text/html", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "root route with aff query returns not found", method: http.MethodGet, target: "/?aff=hZMJ", accept: "text/html", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "head request for frontend route serves spa", method: http.MethodHead, target: "/login?from=home", accept: "*/*", wantStatus: http.StatusOK, wantSPA: true},
		{name: "api docs style frontend route serves spa index", method: http.MethodGet, target: "/api-docs-ui", accept: "text/html", wantStatus: http.StatusOK, wantSPA: true},
		{name: "v10 style frontend route serves spa index", method: http.MethodGet, target: "/v10/overview", accept: "text/html", wantStatus: http.StatusOK, wantSPA: true},
		{name: "api path stays not found", method: http.MethodGet, target: "/api/not-found", accept: "application/json", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "v1 path stays not found", method: http.MethodGet, target: "/v1/not-found", accept: "application/json", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "asset path stays not found", method: http.MethodGet, target: "/assets/not-found.js", accept: "*/*", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "php probe path returns not found", method: http.MethodGet, target: "/wp-admin/setup-config.php", accept: "*/*", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "html probe path returns not found", method: http.MethodGet, target: "/google12345.html", accept: "text/html", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "text probe path returns not found", method: http.MethodGet, target: "/random.txt", accept: "*/*", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "script probe path with query returns not found", method: http.MethodGet, target: "/foo/bar.js?v=1", accept: "*/*", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "dot git config path returns not found", method: http.MethodGet, target: "/.git/config", accept: "*/*", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "dot git head path returns not found", method: http.MethodGet, target: "/.git/HEAD", accept: "*/*", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "dot svn entries path returns not found", method: http.MethodGet, target: "/.svn/entries", accept: "*/*", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "server status path returns not found", method: http.MethodGet, target: "/server-status", accept: "*/*", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "debug path returns not found", method: http.MethodGet, target: "/debug/default/view", accept: "*/*", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "admin path returns not found", method: http.MethodGet, target: "/admin", accept: "text/html", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "json request for unknown path returns not found", method: http.MethodGet, target: "/unknown-endpoint", accept: "application/json", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
		{name: "post request for unknown path returns not found", method: http.MethodPost, target: "/unknown-endpoint", accept: "application/json", wantStatus: http.StatusNotFound, wantSPA: false, wantJSONBody: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			router := gin.New()
			routerpkg.SetWebRouter(router, buildFS, indexPage)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(testCase.method, testCase.target, nil)
			if testCase.accept != "" {
				request.Header.Set("Accept", testCase.accept)
			}
			router.ServeHTTP(recorder, request)

			if recorder.Code != testCase.wantStatus {
				t.Fatalf("expected status %d, got %d", testCase.wantStatus, recorder.Code)
			}

			assertWebSecurityHeaders(t, recorder)

			gotSPA := bytes.Equal(recorder.Body.Bytes(), indexPage)
			if gotSPA != testCase.wantSPA {
				t.Fatalf("expected SPA response %t, got %t for %s", testCase.wantSPA, gotSPA, testCase.target)
			}
			if testCase.wantJSONBody && !bytes.Contains(recorder.Body.Bytes(), []byte(`"error"`)) {
				t.Fatalf("expected JSON error body for %s, got %q", testCase.target, recorder.Body.String())
			}
		})
	}
}
