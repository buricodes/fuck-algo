package httpapi_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func (e *testEnv) checkoutStatus(token, id string) (string, reply) {
	e.t.Helper()
	var data struct {
		Status string `json:"status"`
	}
	r := e.do(http.MethodGet, "/v1/checkout/"+id, token, nil)
	if r.status == http.StatusOK {
		r.into(e.t, &data)
	}
	return data.Status, r
}

func TestCheckoutFlow(t *testing.T) {
	e := newEnv(t)
	u := e.signIn("user_a")
	session := e.startCheckout(u.Token)
	req := e.payments.requests[0]
	if !strings.HasPrefix(session, "cks_") || req.UserID != "user_a" || req.SessionID != session || req.Email != "user_a@example.com" {
		t.Fatalf("checkout did not carry the user, email and session: %+v", e.payments.requests)
	}
	if s, _ := e.checkoutStatus(u.Token, session); s != "pending" {
		t.Fatalf("status before payment = %q", s)
	}

	e.pay("user_a", session)
	if s, _ := e.checkoutStatus(u.Token, session); s != "paid" {
		t.Fatalf("status after payment = %q", s)
	}
	if me := e.me(u.Token); me.Licence.Status != "active" {
		t.Fatalf("user was not activated: %+v", me.Licence)
	}
	for range 4 {
		if r := e.do(http.MethodPost, "/v1/voice/token", u.Token, nil); r.status != http.StatusOK {
			t.Fatalf("active user was limited: %d %s", r.status, r.code())
		}
	}
	if r := e.do(http.MethodPost, "/v1/checkout", u.Token, map[string]any{}); r.status != http.StatusBadRequest || r.code() != "bad_request" {
		t.Fatalf("checkout while active: got %d %s", r.status, r.code())
	}
}

func TestCheckoutWithoutEmail(t *testing.T) {
	e := newEnv(t)
	u := e.signIn("user_a")
	e.users.fail = true
	e.startCheckout(u.Token)
	if got := e.payments.requests[0].Email; got != "" {
		t.Fatalf("email = %q, want none", got)
	}
}

func TestCheckoutStatusBelongsToCaller(t *testing.T) {
	e := newEnv(t)
	owner, other := e.signIn("user_a"), e.signIn("user_b")
	session := e.startCheckout(owner.Token)
	for _, id := range []string{session, "cks_missing", "nope"} {
		if _, r := e.checkoutStatus(other.Token, id); r.status != http.StatusNotFound || r.code() != "not_found" {
			t.Fatalf("%s: got %d %s", id, r.status, r.code())
		}
	}
}

func TestWebhookIdempotentAndVerified(t *testing.T) {
	e := newEnv(t)
	u := e.signIn("user_a")
	session := e.startCheckout(u.Token)
	e.pay("user_a", session)
	if r := e.webhook("msg_" + session); r.status != http.StatusOK {
		t.Fatalf("duplicate webhook: %d %s", r.status, r.code())
	}
	e.payments.event.CheckoutSessionID = "cks_other"
	if r := e.webhook("msg_retry_same_payment"); r.status != http.StatusOK {
		t.Fatalf("same payment, new webhook id: %d %s", r.status, r.code())
	}
	if r := e.send(e.webhookRequest("msg_bad", "v1,forged")); r.status != http.StatusUnauthorized || r.code() != "unauthorised" {
		t.Fatalf("forged webhook: %d %s", r.status, r.code())
	}
}

func TestWebhookActivatesByMetadata(t *testing.T) {
	e := newEnv(t)
	u := e.signIn("user_a")
	e.me(u.Token)
	e.pay("user_a", "cks_made_elsewhere")
	if me := e.me(u.Token); me.Licence.Status != "active" {
		t.Fatalf("user not activated from metadata: %+v", me.Licence)
	}
	e.pay("user_unknown", "cks_nobody")
	if _, err := e.store.UserByClerkID(context.Background(), "user_unknown"); err == nil {
		t.Fatal("a webhook must not create users")
	}
}

func TestWebhookIgnoresOtherEvents(t *testing.T) {
	e := newEnv(t)
	if r := e.webhook("msg_other"); r.status != http.StatusOK {
		t.Fatalf("got %d %s", r.status, r.code())
	}
}

func TestPaymentsUnavailable(t *testing.T) {
	e := newEnv(t)
	e.payments.available = false
	if r := e.do(http.MethodPost, "/v1/checkout", e.signIn("user_a").Token, map[string]any{}); r.status != 503 || r.code() != "unavailable" {
		t.Fatalf("checkout: got %d %s", r.status, r.code())
	}
	if r := e.webhook("msg_1"); r.status != 503 || r.code() != "unavailable" {
		t.Fatalf("webhook: got %d %s", r.status, r.code())
	}
}
