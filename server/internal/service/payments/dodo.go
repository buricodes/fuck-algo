package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/dodopayments/dodopayments-go"
	"github.com/dodopayments/dodopayments-go/option"
	"github.com/dodopayments/dodopayments-go/shared"

	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/svix"
)

const Provider = "dodo"

const EventPaymentSucceeded = "payment.succeeded"

const (
	metaUserID    = "user_id"
	metaSessionID = "session_id"
)

var (
	ErrInvalidSignature = errors.New("payments: invalid webhook signature")
	ErrInvalidPayload   = errors.New("payments: invalid webhook payload")
)

type Settings struct {
	APIKey        string
	WebhookSecret string
	ProductID     string
	WebsiteURL    string
	Live          bool
}

type CheckoutRequest struct {
	SessionID string
	UserID    string
	Email     string
}

type Checkout struct {
	ProviderSessionID string
	URL               string
}

type WebhookHeaders struct {
	ID        string
	Timestamp string
	Signature string
}

type PaymentEvent struct {
	WebhookID         string
	EventType         string
	PaymentID         string
	CheckoutSessionID string
	UserID            string
	Email             string
	Amount            int64
	Currency          string
}

type Dodo struct {
	client   *dodopayments.Client
	settings Settings
}

func NewDodo(s Settings) *Dodo {
	d := &Dodo{settings: s}
	if s.APIKey != "" {
		mode := option.WithEnvironmentTestMode()
		if s.Live {
			mode = option.WithEnvironmentLiveMode()
		}
		d.client = dodopayments.NewClient(option.WithBearerToken(s.APIKey), mode)
	}
	return d
}

func (d *Dodo) Available() bool {
	return d.client != nil && d.settings.ProductID != "" && svix.CheckSecret(d.settings.WebhookSecret) == nil
}

func (d *Dodo) CreateCheckout(ctx context.Context, req CheckoutRequest) (Checkout, error) {
	params := dodopayments.CheckoutSessionRequestParam{
		ProductCart: dodopayments.F([]dodopayments.ProductItemReqParam{{
			ProductID: dodopayments.F(d.settings.ProductID),
			Quantity:  dodopayments.F(int64(1)),
		}}),
		Metadata: dodopayments.F(dodopayments.MetadataParam{
			metaUserID:    shared.UnionString(req.UserID),
			metaSessionID: shared.UnionString(req.SessionID),
		}),
		ReturnURL: dodopayments.F(d.settings.WebsiteURL + "/thanks?session=" + url.QueryEscape(req.SessionID)),
	}
	if req.Email != "" {
		params.Customer = dodopayments.F[dodopayments.CustomerRequestUnionParam](dodopayments.NewCustomerParam{
			Email: dodopayments.F(req.Email),
		})
	}
	res, err := d.client.CheckoutSessions.New(ctx, dodopayments.CheckoutSessionNewParams{CheckoutSessionRequest: params})
	if err != nil {
		return Checkout{}, &apperr.UpstreamError{Provider: Provider, Err: err}
	}
	if res.CheckoutURL == "" {
		return Checkout{}, &apperr.UpstreamError{Provider: Provider, Body: "checkout session had no url"}
	}
	return Checkout{ProviderSessionID: res.SessionID, URL: res.CheckoutURL}, nil
}

type webhookEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type paymentData struct {
	PaymentID   string         `json:"payment_id"`
	TotalAmount int64          `json:"total_amount"`
	Currency    string         `json:"currency"`
	Metadata    map[string]any `json:"metadata"`
	Customer    struct {
		Email string `json:"email"`
	} `json:"customer"`
}

func (d *Dodo) ParseWebhook(body []byte, h WebhookHeaders, now time.Time) (*PaymentEvent, error) {
	if err := svix.Verify(body, h.ID, h.Timestamp, h.Signature, d.settings.WebhookSecret, now); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSignature, err)
	}
	var env webhookEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	if env.Type != EventPaymentSucceeded {
		return nil, nil
	}
	var p paymentData
	if err := json.Unmarshal(env.Data, &p); err != nil || p.PaymentID == "" {
		return nil, fmt.Errorf("%w: payment data: %v", ErrInvalidPayload, err)
	}
	return &PaymentEvent{
		WebhookID:         h.ID,
		EventType:         env.Type,
		PaymentID:         p.PaymentID,
		CheckoutSessionID: metaString(p.Metadata, metaSessionID),
		UserID:            metaString(p.Metadata, metaUserID),
		Email:             p.Customer.Email,
		Amount:            p.TotalAmount,
		Currency:          p.Currency,
	}, nil
}

func metaString(meta map[string]any, key string) string {
	s, _ := meta[key].(string)
	return s
}
