package middleware

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ServeSPA registers fallback SPA serving for React/Vite client-side routing on a Gin engine.
func ServeSPA(r *gin.Engine, staticDir string) {
	if staticDir == "" {
		staticDir = os.Getenv("CUBIT_WEB_DIR")
	}
	if staticDir == "" {
		if _, err := os.Stat("./web/dist"); err == nil {
			staticDir = "./web/dist"
		}
	}
	if staticDir == "" {
		return
	}

	absDir, err := filepath.Abs(staticDir)
	if err != nil {
		return
	}
	if _, err := os.Stat(absDir); err != nil {
		return
	}

	r.NoRoute(func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		// Never serve SPA for API or health endpoints
		if strings.HasPrefix(reqPath, "/api") || reqPath == "/health" {
			c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
			return
		}

		cleanPath := filepath.Clean(reqPath)
		fpath := filepath.Join(absDir, cleanPath)

		info, err := os.Stat(fpath)
		if os.IsNotExist(err) || (err == nil && info.IsDir()) {
			fpath = filepath.Join(absDir, "index.html")
		}

		data, err := os.ReadFile(fpath)
		if err != nil {
			c.String(http.StatusNotFound, "404 page not found")
			return
		}

		c.Writer.Header().Set("Content-Type", getMimeType(fpath))
		c.Writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
		c.Data(http.StatusOK, getMimeType(fpath), data)
	})
}

func getMimeType(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}
