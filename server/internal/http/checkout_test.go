package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

type statusData struct {
	Status     string  `json:"status"`
	LicenceKey *string `json:"licenceKey"`
}

func (e *testEnv) checkoutStatus(id string) (statusData, reply) {
	e.t.Helper()
	var data statusData
	r := e.do(http.MethodGet, "/v1/checkout/"+id, "", nil)
	if r.status == http.StatusOK {
		r.into(e.t, &data)
	}
	return data, r
}

func TestCheckoutFlow(t *testing.T) {
	e := newEnv(t)
	in := e.install()
	session := e.startCheckout(&in.InstallID)
	if !strings.HasPrefix(session, "cks_") || e.payments.requests[0].InstallID != in.InstallID || e.payments.requests[0].SessionID != session {
		t.Fatalf("checkout did not carry the install and session: %+v", e.payments.requests)
	}
	if s, _ := e.checkoutStatus(session); s.Status != "pending" || s.LicenceKey != nil {
		t.Fatalf("status before payment = %+v", s)
	}

	key := e.buyLicence(in.InstallID, session)
	s, _ := e.checkoutStatus(session)
	if s.Status != "paid" || s.LicenceKey == nil || *s.LicenceKey != key {
		t.Fatalf("status after payment = %+v", s)
	}
	me := e.me(in.Token)
	if me.Licence.Status != "active" || me.Licence.KeyHint == nil || *me.Licence.KeyHint != key[len(key)-4:] {
		t.Fatalf("install was not activated: %+v", me.Licence)
	}

	for range 4 {
		if r := e.do(http.MethodPost, "/v1/voice/token", in.Token, nil); r.status != http.StatusOK {
			t.Fatalf("licensed install was limited: %d %s", r.status, r.code())
		}
	}

	e.now = e.now.Add(25 * time.Hour)
	if s, _ := e.checkoutStatus(session); s.Status != "paid" || s.LicenceKey != nil {
		t.Fatalf("key still shown after 24 hours: %+v", s)
	}
}

func TestWebhookIdempotentAndVerified(t *testing.T) {
	e := newEnv(t)
	session := e.startCheckout(nil)
	key := e.buyLicence("", session)
	if r := e.webhook("msg_" + session); r.status != http.StatusOK {
		t.Fatalf("duplicate webhook: %d %s", r.status, r.code())
	}
	if s, _ := e.checkoutStatus(session); s.LicenceKey == nil || *s.LicenceKey != key {
		t.Fatalf("duplicate webhook changed the key")
	}
	e.payments.event.CheckoutSessionID = "cks_other"
	if r := e.webhook("msg_retry_same_payment"); r.status != http.StatusOK {
		t.Fatalf("same payment, new webhook id: %d %s", r.status, r.code())
	}

	req := e.webhookRequest("msg_bad", "v1,forged")
	if r := e.send(req); r.status != http.StatusUnauthorized || r.code() != "unauthorised" {
		t.Fatalf("forged webhook: %d %s", r.status, r.code())
	}
}

func TestWebhookIgnoresOtherEvents(t *testing.T) {
	e := newEnv(t)
	if r := e.webhook("msg_other"); r.status != http.StatusOK {
		t.Fatalf("got %d %s", r.status, r.code())
	}
}

func TestCheckoutValidation(t *testing.T) {
	tests := []struct {
		name   string
		body   map[string]any
		status int
		code   string
	}{
		{"bad install id", map[string]any{"installId": "nope"}, 400, "bad_request"},
		{"unknown install", map[string]any{"installId": "6f1c2b8e-3d4a-4f5b-9c6d-7e8f9a0b1c2d"}, 404, "not_found"},
		{"bad email", map[string]any{"installId": nil, "email": "not an email"}, 400, "bad_request"},
		{"email with name", map[string]any{"email": "Aman <a@example.com>"}, 400, "bad_request"},
		{"good email", map[string]any{"email": "a@example.com"}, 200, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newEnv(t).do(http.MethodPost, "/v1/checkout", "", tt.body)
			if r.status != tt.status || r.code() != tt.code {
				t.Fatalf("got %d %s, want %d %s", r.status, r.code(), tt.status, tt.code)
			}
		})
	}
}

func TestCheckoutStatusNotFound(t *testing.T) {
	e := newEnv(t)
	for _, id := range []string{"cks_missing", "nope"} {
		if _, r := e.checkoutStatus(id); r.status != http.StatusNotFound || r.code() != "not_found" {
			t.Fatalf("%s: got %d %s", id, r.status, r.code())
		}
	}
}

func TestPaymentsUnavailable(t *testing.T) {
	e := newEnv(t)
	e.payments.available = false
	if r := e.do(http.MethodPost, "/v1/checkout", "", map[string]any{}); r.status != 503 || r.code() != "unavailable" {
		t.Fatalf("checkout: got %d %s", r.status, r.code())
	}
	if r := e.webhook("msg_1"); r.status != 503 || r.code() != "unavailable" {
		t.Fatalf("webhook: got %d %s", r.status, r.code())
	}
}

func TestLicenceActivate(t *testing.T) {
	e := newEnv(t)
	key := e.buyLicence("", e.startCheckout(nil))
	typed := strings.ToLower(strings.ReplaceAll(key, "-", " "))

	for i := range 6 {
		in := e.install()
		r := e.do(http.MethodPost, "/v1/licence/activate", in.Token, map[string]string{"key": typed})
		if i < 5 {
			if r.status != http.StatusOK {
				t.Fatalf("install %d: got %d %s", i, r.status, r.code())
			}
			if again := e.do(http.MethodPost, "/v1/licence/activate", in.Token, map[string]string{"key": key}); again.status != http.StatusOK {
				t.Fatalf("re-activating the same install should succeed, got %d %s", again.status, again.code())
			}
			continue
		}
		if r.status != http.StatusConflict || r.code() != "licence_full" {
			t.Fatalf("sixth install: got %d %s", r.status, r.code())
		}
	}
}

func TestLicenceActivateInvalid(t *testing.T) {
	e := newEnv(t)
	in := e.install()
	for _, key := range []string{"", "FA-AAAA-AAAA-AAAA-AAAA", "garbage", strings.Repeat("A", 100)} {
		r := e.do(http.MethodPost, "/v1/licence/activate", in.Token, map[string]string{"key": key})
		if r.status != http.StatusBadRequest || r.code() != "licence_invalid" {
			t.Fatalf("key %q: got %d %s", key, r.status, r.code())
		}
	}
}
