package router

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

func shouldRedirectDocViewer(c *gin.Context) bool {
	if c.Query("raw") == "1" {
		return false
	}

	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return false
	}

	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "text/html") {
		return true
	}

	if c.GetHeader("Sec-Fetch-Dest") == "document" {
		return true
	}

	if c.GetHeader("Upgrade-Insecure-Requests") == "1" {
		return true
	}

	return false
}

func registerDocViewerRedirect(router *gin.Engine, path string, target string, docsFS fs.FS) {
	handler := func(c *gin.Context) {
		if shouldRedirectDocViewer(c) {
			c.Redirect(http.StatusTemporaryRedirect, target)
			return
		}
		c.FileFromFS(strings.TrimPrefix(path, "/"), http.FS(docsFS))
	}

	router.GET(path, handler)
	router.HEAD(path, handler)
}

func SetWebRouter(router *gin.Engine, buildFS embed.FS, indexPage []byte) {
	docsFS, err := fs.Sub(buildFS, "web/dist")
	if err != nil {
		panic(err)
	}

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	registerDocViewerRedirect(router, "/docs/openapi/api.json", "/docs-viewer/openapi?spec=%2Fdocs%2Fopenapi%2Fapi.json&title=API%20Documentation", docsFS)
	registerDocViewerRedirect(router, "/docs/openapi/relay.json", "/docs-viewer/openapi?spec=%2Fdocs%2Fopenapi%2Frelay.json&title=Relay%20Documentation", docsFS)
	router.Use(static.Serve("/", common.EmbedFolder(buildFS, "web/dist")))
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") {
			controller.RelayNotFound(c)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexPage)
	})
}
