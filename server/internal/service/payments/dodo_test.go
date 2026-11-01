package payments

import (
	"encoding/base64"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/buricodes/fuck-algo/server/pkg/svix"
)

func TestParseWebhook(t *testing.T) {
	key := []byte("dodo-test-signing-key")
	d := NewDodo(Settings{WebhookSecret: "whsec_" + base64.StdEncoding.EncodeToString(key)})
	now := time.Unix(1_800_000_000, 0)
	ts := strconv.FormatInt(now.Unix(), 10)
	sign := func(body string) WebhookHeaders {
		sig := base64.StdEncoding.EncodeToString(svix.Sign(key, "msg_1", ts, []byte(body)))
		return WebhookHeaders{ID: "msg_1", Timestamp: ts, Signature: "v1," + sig}
	}
	paid := `{"business_id":"bus_1","type":"payment.succeeded","timestamp":"2026-11-01T10:00:00Z","data":{
		"payload_type":"Payment","payment_id":"pay_1","total_amount":500,"currency":"USD",
		"customer":{"customer_id":"cus_1","email":"a@example.com","name":"A"},
		"metadata":{"user_id":"user_2abc","session_id":"cks_abc","other":3}}}`

	tests := []struct {
		name    string
		body    string
		headers WebhookHeaders
		want    *PaymentEvent
		err     error
	}{
		{"payment succeeded", paid, sign(paid), &PaymentEvent{
			WebhookID: "msg_1", EventType: EventPaymentSucceeded, PaymentID: "pay_1", CheckoutSessionID: "cks_abc",
			UserID: "user_2abc", Email: "a@example.com", Amount: 500, Currency: "USD",
		}, nil},
		{"other event ignored", `{"type":"payment.failed","data":{}}`, sign(`{"type":"payment.failed","data":{}}`), nil, nil},
		{"forged", paid, WebhookHeaders{ID: "msg_1", Timestamp: ts, Signature: "v1,Zm9yZ2Vk"}, nil, ErrInvalidSignature},
		{"missing payment id", `{"type":"payment.succeeded","data":{}}`, sign(`{"type":"payment.succeeded","data":{}}`), nil, ErrInvalidPayload},
		{"not json", `nope`, sign(`nope`), nil, ErrInvalidPayload},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.ParseWebhook([]byte(tt.body), tt.headers, now)
			if !errors.Is(err, tt.err) {
				t.Fatalf("err = %v, want %v", err, tt.err)
			}
			if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
				t.Fatalf("event = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAvailable(t *testing.T) {
	secret := "whsec_" + base64.StdEncoding.EncodeToString([]byte("k"))
	tests := []struct {
		name string
		s    Settings
		want bool
	}{
		{"complete", Settings{APIKey: "k", WebhookSecret: secret, ProductID: "pdt_1"}, true},
		{"no api key", Settings{WebhookSecret: secret, ProductID: "pdt_1"}, false},
		{"no product", Settings{APIKey: "k", WebhookSecret: secret}, false},
		{"bad secret", Settings{APIKey: "k", WebhookSecret: "nope!", ProductID: "pdt_1"}, false},
	}
	for _, tt := range tests {
		if got := NewDodo(tt.s).Available(); got != tt.want {
			t.Errorf("%s: Available() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
