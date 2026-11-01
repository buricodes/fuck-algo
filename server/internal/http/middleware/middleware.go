package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/response"
)

const extensionScheme = "chrome-extension://"

type CORSOptions struct {
	WebsiteURL        string
	ExtensionOrigins  []string
	AnyExtensionInDev bool
}

func (o CORSOptions) allowed(origin string) bool {
	switch {
	case origin == "":
		return false
	case origin == o.WebsiteURL:
		return true
	case !strings.HasPrefix(origin, extensionScheme):
		return false
	case len(o.ExtensionOrigins) == 0:
		return o.AnyExtensionInDev
	}
	return slices.Contains(o.ExtensionOrigins, origin)
}

func CORS(opts CORSOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if opts.allowed(origin) {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			h.Set("Access-Control-Max-Age", "600")
		}
		c.Writer.Header().Add("Vary", "Origin")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func BodyLimit(limitFor func(route string) int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := limitFor(c.FullPath())
		if c.Request.ContentLength > limit {
			response.Fail(c, TooLarge())
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

func TooLarge() error {
	return apperr.New(apperr.BadRequest, "The request body is too large.").WithStatus(http.StatusRequestEntityTooLarge)
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.InfoContext(c.Request.Context(), "request",
			"method", c.Request.Method,
			"route", c.FullPath(),
			"status", c.Writer.Status(),
			"ms", time.Since(start).Milliseconds(),
		)
	}
}

func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		slog.ErrorContext(c.Request.Context(), "panic recovered", "route", c.FullPath(), "panic", recovered)
		response.Fail(c, apperr.New(apperr.Internal, "Something went wrong on our side."))
	})
}
