package router

import (
	"embed"
	"path"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

var blockedWebRoutePrefixes = map[string]struct{}{
	"admin":         {},
	"administrator": {},
	"actuator":      {},
	"debug":         {},
	"phpinfo":       {},
	"server-status": {},
	"wp-admin":      {},
	"wp-login":      {},
}

func SetWebRouter(router *gin.Engine, buildFS embed.FS, indexPage []byte) {
	router.Use(middleware.SecurityHeaders())
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	router.Use(static.Serve("/", common.EmbedFolder(buildFS, "web/dist")))
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		requestPath := c.Request.URL.Path
		if strings.HasPrefix(requestPath, "/assets") {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private, max-age=0")
			controller.RelayNotFound(c)
			return
		}
		if isExactWebNamespace(requestPath, "v1") || isExactWebNamespace(requestPath, "api") {
			controller.RelayNotFound(c)
			return
		}
		if shouldReturnNotFoundForWebRequest(c.Request.Method, c.GetHeader("Accept"), requestPath, c.Request.URL.Query().Has("aff")) {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private, max-age=0")
			controller.RelayNotFound(c)
			return
		}
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private, max-age=0")
		c.Header("Clear-Site-Data", "\"cache\"")
		c.Data(200, "text/html; charset=utf-8", indexPage)
	})
}

func isExactWebNamespace(requestPath string, namespace string) bool {
	prefix := "/" + namespace
	return requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/")
}

func shouldReturnNotFoundForWebRequest(method string, accept string, requestPath string, hasAffQuery bool) bool {
	if method != "" && method != "GET" && method != "HEAD" {
		return true
	}
	if accept != "" && !strings.Contains(accept, "text/html") && !strings.Contains(accept, "*/*") {
		return true
	}
	if hasAffQuery {
		return true
	}
	lastSegment := path.Base(requestPath)
	if lastSegment == "." || lastSegment == "/" || lastSegment == "" {
		return false
	}
	if strings.HasPrefix(lastSegment, ".") {
		return true
	}
	ext := path.Ext(lastSegment)
	if ext != "" {
		return true
	}
	trimmedPath := strings.Trim(requestPath, "/")
	if trimmedPath == "" {
		return false
	}
	for _, segment := range strings.Split(trimmedPath, "/") {
		if strings.HasPrefix(segment, ".") {
			return true
		}
	}
	firstSegment := strings.SplitN(trimmedPath, "/", 2)[0]
	_, blocked := blockedWebRoutePrefixes[firstSegment]
	if blocked {
		return true
	}
	return false
}
