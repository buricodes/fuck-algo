package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/service/payments"
	"github.com/buricodes/fuck-algo/server/internal/store"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/response"
)

const (
	maxSessionIDChars = 64
	sessionPrefix     = "cks_"
	sessionIDBytes    = 18
)

type checkoutResponse struct {
	SessionID   string `json:"sessionId"`
	CheckoutURL string `json:"checkoutUrl"`
}

type checkoutStatusResponse struct {
	Status string `json:"status"`
}

func (h *handler) createCheckout(c *gin.Context) {
	if !h.Payments.Available() {
		response.Fail(c, unavailable("Checkout"))
		return
	}
	res, err := h.startCheckout(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, res)
}

func (h *handler) startCheckout(c *gin.Context) (checkoutResponse, error) {
	ctx, u := c.Request.Context(), currentUser(c)
	if u.Active() {
		return checkoutResponse{}, badRequest("You already have the full version.")
	}
	if err := h.ensureEmail(ctx, &u); err != nil {
		return checkoutResponse{}, err
	}
	id, err := newSessionID()
	if err != nil {
		return checkoutResponse{}, err
	}
	session := store.CheckoutSession{ID: id, UserID: u.ID, Status: store.CheckoutPending, CreatedAt: h.Now()}
	if err := h.Store.CreateCheckoutSession(ctx, session); err != nil {
		return checkoutResponse{}, err
	}
	co, err := h.Payments.CreateCheckout(ctx, payments.CheckoutRequest{SessionID: id, UserID: u.ClerkID, Email: u.Email})
	if err != nil {
		return checkoutResponse{}, err
	}
	if err := h.Store.SetCheckoutProviderID(ctx, id, co.ProviderSessionID); err != nil {
		return checkoutResponse{}, err
	}
	return checkoutResponse{SessionID: id, CheckoutURL: co.URL}, nil
}

func (h *handler) checkoutStatus(c *gin.Context) {
	id := c.Param("sessionId")
	if !strings.HasPrefix(id, sessionPrefix) || len(id) > maxSessionIDChars {
		response.Fail(c, checkoutNotFound())
		return
	}
	s, err := h.Store.CheckoutSession(c.Request.Context(), id)
	if errors.Is(err, store.ErrNotFound) || (err == nil && s.UserID != currentUser(c).ID) {
		err = checkoutNotFound()
	}
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, checkoutStatusResponse{Status: s.Status})
}

func newSessionID() (string, error) {
	raw := make([]byte, sessionIDBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("reading randomness: %w", err)
	}
	return sessionPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

func checkoutNotFound() error {
	return apperr.New(apperr.NotFound, "That checkout does not exist.")
}
