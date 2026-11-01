package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/config"
	"github.com/buricodes/fuck-algo/server/internal/domain"
	"github.com/buricodes/fuck-algo/server/internal/http/middleware"
	"github.com/buricodes/fuck-algo/server/internal/service/checks"
	"github.com/buricodes/fuck-algo/server/internal/service/payments"
	"github.com/buricodes/fuck-algo/server/internal/service/voice"
	"github.com/buricodes/fuck-algo/server/internal/store"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
)

type STT interface {
	Available() bool
	RealtimeToken(ctx context.Context) (string, error)
	Transcribe(ctx context.Context, audio []byte, filename string) (string, error)
}

type Payments interface {
	Available() bool
	CreateCheckout(ctx context.Context, req payments.CheckoutRequest) (payments.Checkout, error)
	ParseWebhook(body []byte, h payments.WebhookHeaders, now time.Time) (*payments.PaymentEvent, error)
}

type Deps struct {
	Config   *config.Config
	Store    store.Store
	Auth     Verifier
	Users    Directory
	STT      STT
	Checks   *checks.Service
	Voice    *voice.Service
	Payments Payments
	Now      func() time.Time
}

type handler struct {
	Deps
}

func (h *handler) trialLimit(c store.Counter) int {
	if c == store.CounterVoice {
		return h.Config.Trial.Voice
	}
	return h.Config.Trial.Checks
}

func (h *handler) licenceOf(u store.User) domain.Licence {
	l := domain.Licence{
		Status: domain.LicenceTrial,
		Trial: domain.Trial{
			VoiceLeft:  max(0, h.Config.Trial.Voice-u.VoiceUsed),
			ChecksLeft: max(0, h.Config.Trial.Checks-u.ChecksUsed),
		},
	}
	if u.Active() {
		l.Status = domain.LicenceActive
	}
	return l
}

func (h *handler) spend(ctx context.Context, u store.User, c store.Counter, n int) (int, error) {
	if u.Active() || n == 0 {
		return n, nil
	}
	granted, err := h.Store.SpendTrial(ctx, u.ID, c, n, h.trialLimit(c))
	if errors.Is(err, store.ErrTrialSpent) {
		return 0, apperr.New(apperr.TrialExhausted, trialMessage(c))
	}
	return granted, err
}

func (h *handler) refund(ctx context.Context, u store.User, c store.Counter, n int) {
	if u.Active() || n <= 0 {
		return
	}
	if err := h.Store.RefundTrial(context.WithoutCancel(ctx), u.ID, c, n); err != nil {
		slog.ErrorContext(ctx, "refunding trial failed", "user", u.ID, "error", err)
	}
}

func trialMessage(c store.Counter) string {
	if c == store.CounterVoice {
		return "Your free voice rules are used up."
	}
	return "Your free AI checks are used up."
}

func bindJSON(c *gin.Context, dst any) error {
	if err := json.NewDecoder(c.Request.Body).Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return middleware.TooLarge()
		}
		return badRequest("The request body is not valid JSON.")
	}
	return nil
}

func badRequest(message string) error { return apperr.New(apperr.BadRequest, message) }

func unavailable(what string) error {
	return apperr.New(apperr.Unavailable, what+" is not available right now.")
}
