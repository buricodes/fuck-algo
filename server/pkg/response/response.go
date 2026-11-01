package response

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/pkg/apperr"
)

type errorBody struct {
	Code    apperr.Code `json:"code"`
	Message string      `json:"message"`
}

type envelope struct {
	OK    bool       `json:"ok"`
	Data  any        `json:"data,omitempty"`
	Error *errorBody `json:"error,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, envelope{OK: true, Data: data})
}

func Fail(c *gin.Context, err error) {
	e := apperr.From(err)
	if e.Code == apperr.Internal || e.Code == apperr.Upstream {
		slog.ErrorContext(c.Request.Context(), "request failed", "path", c.FullPath(), "code", e.Code, "error", err)
	}
	c.AbortWithStatusJSON(e.Status, envelope{Error: &errorBody{Code: e.Code, Message: e.Message}})
}
