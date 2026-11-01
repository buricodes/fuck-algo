package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/http/middleware"
	"github.com/buricodes/fuck-algo/server/internal/service/licence"
	"github.com/buricodes/fuck-algo/server/internal/service/payments"
	"github.com/buricodes/fuck-algo/server/internal/store"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/response"
)

type webhookResponse struct {
	Received bool `json:"received"`
}

func (h *handler) dodoWebhook(c *gin.Context) {
	if !h.Payments.Available() {
		response.Fail(c, unavailable("Payments"))
		return
	}
	event, err := h.parseWebhook(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if event != nil {
		err = h.recordPayment(c, event)
	}
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, webhookResponse{Received: true})
}

func (h *handler) parseWebhook(c *gin.Context) (*payments.PaymentEvent, error) {
	body, err := io.ReadAll(c.Request.Body)
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return nil, middleware.TooLarge()
	}
	if err != nil {
		return nil, badRequest("Could not read the webhook body.")
	}
	event, err := h.Payments.ParseWebhook(body, payments.WebhookHeaders{
		ID:        c.GetHeader("webhook-id"),
		Timestamp: c.GetHeader("webhook-timestamp"),
		Signature: c.GetHeader("webhook-signature"),
	}, h.Now())
	switch {
	case errors.Is(err, payments.ErrInvalidSignature):
		slog.WarnContext(c.Request.Context(), "webhook signature rejected", "error", err)
		return nil, apperr.New(apperr.Unauthorised, "Invalid webhook signature.")
	case errors.Is(err, payments.ErrInvalidPayload):
		return nil, badRequest("Invalid webhook payload.")
	}
	return event, err
}

func (h *handler) recordPayment(c *gin.Context, event *payments.PaymentEvent) error {
	key, err := licence.NewKey()
	if err != nil {
		return err
	}
	installID := event.InstallID
	if !licence.IsUUID(installID) {
		installID = ""
	}
	err = h.Store.RecordPayment(c.Request.Context(), store.Payment{
		Provider:          payments.Provider,
		WebhookID:         event.WebhookID,
		EventType:         event.EventType,
		ProviderPaymentID: event.PaymentID,
		CheckoutSessionID: event.CheckoutSessionID,
		InstallID:         installID,
		Email:             event.Email,
		Amount:            event.Amount,
		Currency:          event.Currency,
		LicenceID:         licence.NewUUID(),
		KeyHash:           licence.HashKey(key),
		KeyHint:           licence.Hint(key),
		RevealKey:         key,
		MaxInstalls:       h.Config.Licence.MaxInstalls,
		At:                h.Now(),
	})
	if errors.Is(err, store.ErrDuplicate) {
		slog.InfoContext(c.Request.Context(), "duplicate webhook ignored", "webhookId", event.WebhookID)
		return nil
	}
	if err == nil {
		slog.InfoContext(c.Request.Context(), "licence issued", "payment", event.PaymentID, "session", event.CheckoutSessionID, "hasInstall", installID != "")
	}
	return err
}
