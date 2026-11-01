package httpapi

import (
	"errors"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/service/licence"
	"github.com/buricodes/fuck-algo/server/internal/service/payments"
	"github.com/buricodes/fuck-algo/server/internal/store"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/response"
)

const (
	maxEmailChars     = 254
	maxSessionIDChars = 64
	sessionPrefix     = "cks_"
)

type checkoutRequest struct {
	InstallID *string `json:"installId"`
	Email     string  `json:"email"`
}

type checkoutResponse struct {
	SessionID   string `json:"sessionId"`
	CheckoutURL string `json:"checkoutUrl"`
}

type checkoutStatusResponse struct {
	Status     string  `json:"status"`
	LicenceKey *string `json:"licenceKey"`
}

func (h *handler) createCheckout(c *gin.Context) {
	if !h.Payments.Available() {
		response.Fail(c, unavailable("Checkout"))
		return
	}
	var req checkoutRequest
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	res, err := h.startCheckout(c, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, res)
}

func (h *handler) startCheckout(c *gin.Context, req checkoutRequest) (checkoutResponse, error) {
	ctx := c.Request.Context()
	email, err := checkoutEmail(req.Email)
	if err != nil {
		return checkoutResponse{}, err
	}
	if err := h.checkInstall(c, req.InstallID); err != nil {
		return checkoutResponse{}, err
	}
	id, err := licence.NewCheckoutSessionID()
	if err != nil {
		return checkoutResponse{}, err
	}
	session := store.CheckoutSession{ID: id, InstallID: req.InstallID, Status: store.CheckoutPending, CreatedAt: h.Now()}
	if err := h.Store.CreateCheckoutSession(ctx, session); err != nil {
		return checkoutResponse{}, err
	}
	co, err := h.Payments.CreateCheckout(ctx, payments.CheckoutRequest{SessionID: id, InstallID: deref(req.InstallID), Email: email})
	if err != nil {
		return checkoutResponse{}, err
	}
	if err := h.Store.SetCheckoutProviderID(ctx, id, co.ProviderSessionID); err != nil {
		return checkoutResponse{}, err
	}
	return checkoutResponse{SessionID: id, CheckoutURL: co.URL}, nil
}

func (h *handler) checkInstall(c *gin.Context, installID *string) error {
	if installID == nil {
		return nil
	}
	if !licence.IsUUID(*installID) {
		return badRequest("installId must be a UUID or null.")
	}
	exists, err := h.Store.InstallExists(c.Request.Context(), *installID)
	if err == nil && !exists {
		return apperr.New(apperr.NotFound, "That install is not recognised.")
	}
	return err
}

func checkoutEmail(raw string) (string, error) {
	email := strings.TrimSpace(raw)
	if email == "" {
		return "", nil
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > maxEmailChars {
		return "", badRequest("That email address does not look right.")
	}
	return email, nil
}

func (h *handler) checkoutStatus(c *gin.Context) {
	id := c.Param("sessionId")
	if !strings.HasPrefix(id, sessionPrefix) || len(id) > maxSessionIDChars {
		response.Fail(c, checkoutNotFound())
		return
	}
	s, err := h.Store.CheckoutSession(c.Request.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		err = checkoutNotFound()
	}
	if err != nil {
		response.Fail(c, err)
		return
	}
	res := checkoutStatusResponse{Status: s.Status}
	if s.Status == store.CheckoutPaid && h.Now().Sub(s.CreatedAt) < h.Config.Licence.RevealWindow() {
		res.LicenceKey = s.RevealKey
	}
	response.OK(c, res)
}

func checkoutNotFound() error {
	return apperr.New(apperr.NotFound, "That checkout does not exist.")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
