package httpapi

import (
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/domain"
	"github.com/buricodes/fuck-algo/server/internal/service/licence"
	"github.com/buricodes/fuck-algo/server/internal/store"
	"github.com/buricodes/fuck-algo/server/pkg/response"
)

const maxClientFieldChars = 40

type createInstallRequest struct {
	Version string `json:"version"`
	Browser string `json:"browser"`
}

type installResponse struct {
	InstallID string         `json:"installId"`
	Token     string         `json:"token,omitempty"`
	Licence   domain.Licence `json:"licence"`
}

func (h *handler) createInstall(c *gin.Context) {
	var req createInstallRequest
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	req.Version, req.Browser = strings.TrimSpace(req.Version), strings.ToLower(strings.TrimSpace(req.Browser))
	if !clientField(req.Version) || !clientField(req.Browser) {
		response.Fail(c, badRequest("version and browser must each be 1 to 40 characters."))
		return
	}
	token, err := licence.NewInstallToken()
	if err != nil {
		response.Fail(c, err)
		return
	}
	now := h.Now()
	in := store.Install{ID: licence.NewUUID(), TokenHash: licence.HashToken(token), Version: req.Version, Browser: req.Browser, CreatedAt: now, LastSeenAt: now}
	if err := h.Store.CreateInstall(c.Request.Context(), in); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, installResponse{InstallID: in.ID, Token: token, Licence: h.licenceOf(in)})
}

func (h *handler) me(c *gin.Context) {
	in := install(c)
	if err := h.Store.TouchInstall(c.Request.Context(), in.ID, h.Now()); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, installResponse{InstallID: in.ID, Licence: h.licenceOf(in)})
}

func clientField(s string) bool {
	return s != "" && utf8.RuneCountInString(s) <= maxClientFieldChars
}
