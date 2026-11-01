package httpapi

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/domain"
	"github.com/buricodes/fuck-algo/server/internal/service/licence"
	"github.com/buricodes/fuck-algo/server/internal/store"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/response"
)

const maxKeyInputChars = 64

type activateRequest struct {
	Key string `json:"key"`
}

type licenceResponse struct {
	Licence domain.Licence `json:"licence"`
}

func (h *handler) activateLicence(c *gin.Context) {
	var req activateRequest
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	in, err := h.activate(c, req.Key)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, licenceResponse{Licence: h.licenceOf(in)})
}

func (h *handler) activate(c *gin.Context, key string) (store.Install, error) {
	in := install(c)
	canonical, err := licence.NormaliseKey(key)
	if len(key) > maxKeyInputChars || err != nil {
		return in, invalidLicence()
	}
	l, err := h.Store.LicenceByKeyHash(c.Request.Context(), licence.HashKey(canonical))
	if errors.Is(err, store.ErrNotFound) || (err == nil && l.Status != store.LicenceStatusActive) {
		return in, invalidLicence()
	}
	if err != nil {
		return in, err
	}
	err = h.Store.AttachLicence(c.Request.Context(), in.ID, l.ID, h.Config.Licence.MaxInstalls)
	if errors.Is(err, store.ErrLicenceFull) {
		return in, apperr.New(apperr.LicenceFull, "This key is already used on the maximum number of browsers.")
	}
	if err != nil {
		return in, err
	}
	hint := l.KeyHint
	in.LicenceID, in.LicenceActive, in.KeyHint = &l.ID, true, &hint
	return in, nil
}

func invalidLicence() error {
	return apperr.New(apperr.LicenceInvalid, "That licence key is not valid. Check it and try again.")
}
