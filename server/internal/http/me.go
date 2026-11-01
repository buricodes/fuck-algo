package httpapi

import (
	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/domain"
	"github.com/buricodes/fuck-algo/server/pkg/response"
)

type meResponse struct {
	UserID  string         `json:"userId"`
	Email   string         `json:"email"`
	Licence domain.Licence `json:"licence"`
}

func (h *handler) me(c *gin.Context) {
	ctx, u := c.Request.Context(), currentUser(c)
	if err := h.ensureEmail(ctx, &u); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Store.TouchUser(ctx, u.ID, h.Now()); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, meResponse{UserID: u.ClerkID, Email: u.Email, Licence: h.licenceOf(u)})
}
