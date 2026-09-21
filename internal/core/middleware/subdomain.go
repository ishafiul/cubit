package middleware

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
)

// WorkerInvoker defines the minimal interface required for subdomain routing.
type WorkerInvoker interface {
	GetApplicationBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error)
	InvokeApplication(ctx context.Context, appID, method, uri string, headers map[string]string, body []byte) (int, map[string]string, []byte, error)
}

// SubdomainRouter proxies requests matching *.localhost or *.cubit.local to the deployed worker isolate.
func SubdomainRouter(invoker WorkerInvoker) gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}

		var subdomain string
		if strings.HasSuffix(host, ".localhost") {
			subdomain = strings.TrimSuffix(host, ".localhost")
		} else if strings.HasSuffix(host, ".cubit.local") {
			subdomain = strings.TrimSuffix(host, ".cubit.local")
		}

		if subdomain != "" && subdomain != "api" && subdomain != "dashboard" && subdomain != "localhost" {
			app, err := invoker.GetApplicationBySubdomain(c.Request.Context(), subdomain)
			if err == nil && app != nil {
				if app.ActiveDeploymentID == "" {
					c.String(http.StatusServiceUnavailable, fmt.Sprintf("Application %q has no active deployment", app.Name))
					c.Abort()
					return
				}

				reqBody, _ := io.ReadAll(c.Request.Body)
				headers := make(map[string]string)
				for k, v := range c.Request.Header {
					if len(v) > 0 {
						headers[k] = v[0]
					}
				}

				status, respHeaders, respBody, err := invoker.InvokeApplication(c.Request.Context(), app.ID, c.Request.Method, c.Request.URL.RequestURI(), headers, reqBody)
				if err != nil {
					c.String(http.StatusInternalServerError, fmt.Sprintf("Worker invocation error: %v", err))
					c.Abort()
					return
				}

				for k, v := range respHeaders {
					c.Writer.Header().Set(k, v)
				}
				c.Data(status, c.Writer.Header().Get("Content-Type"), respBody)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
