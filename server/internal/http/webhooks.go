package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/http/middleware"
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
	ctx := c.Request.Context()
	matched, err := h.Store.RecordPayment(ctx, store.Payment{
		Provider:          payments.Provider,
		WebhookID:         event.WebhookID,
		EventType:         event.EventType,
		ProviderPaymentID: event.PaymentID,
		CheckoutSessionID: event.CheckoutSessionID,
		ClerkID:           event.UserID,
		Amount:            event.Amount,
		Currency:          event.Currency,
		At:                h.Now(),
	})
	switch {
	case errors.Is(err, store.ErrDuplicate):
		slog.InfoContext(ctx, "duplicate webhook ignored", "webhookId", event.WebhookID)
		return nil
	case err != nil:
		return err
	case !matched:
		slog.WarnContext(ctx, "payment recorded but matched no user", "payment", event.PaymentID, "session", event.CheckoutSessionID)
	default:
		slog.InfoContext(ctx, "payment recorded, user active", "payment", event.PaymentID, "session", event.CheckoutSessionID)
	}
	return nil
}
