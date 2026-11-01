package httpapi

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/service/licence"
	"github.com/buricodes/fuck-algo/server/internal/store"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/response"
)

const installKey = "install"

func (h *handler) requireInstall(c *gin.Context) {
	token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !ok || !strings.HasPrefix(token, licence.TokenPrefix) {
		response.Fail(c, unauthorised())
		return
	}
	in, err := h.Store.InstallByTokenHash(c.Request.Context(), licence.HashToken(strings.TrimSpace(token)))
	if errors.Is(err, store.ErrNotFound) {
		err = unauthorised()
	}
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Set(installKey, in)
	c.Next()
}

func install(c *gin.Context) store.Install {
	return c.MustGet(installKey).(store.Install)
}

func byInstall(c *gin.Context) string { return install(c).ID }

func unauthorised() error {
	return apperr.New(apperr.Unauthorised, "This install is not recognised. Reinstall the extension.")
}
