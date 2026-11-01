package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/config"
	httpapi "github.com/buricodes/fuck-algo/server/internal/http"
	"github.com/buricodes/fuck-algo/server/internal/service/checks"
	"github.com/buricodes/fuck-algo/server/internal/service/payments"
	"github.com/buricodes/fuck-algo/server/internal/service/voice"
	"github.com/buricodes/fuck-algo/server/internal/store"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
)

type fakeModel struct {
	available bool
	calls     int
	lastUser  string
	reply     func(user string) (string, error)
}

func (m *fakeModel) Available() bool { return m.available }

func (m *fakeModel) JSON(_ context.Context, _, user string, _ json.RawMessage) ([]byte, error) {
	m.calls++
	m.lastUser = user
	out, err := m.reply(user)
	return []byte(out), err
}

type fakeSTT struct {
	available bool
	fail      bool
}

func (s *fakeSTT) Available() bool { return s.available }

func (s *fakeSTT) RealtimeToken(context.Context) (string, error) {
	if s.fail {
		return "", &apperr.UpstreamError{Provider: "elevenlabs", Status: 500}
	}
	return "sutkn_test", nil
}

func (s *fakeSTT) Transcribe(_ context.Context, audio []byte, filename string) (string, error) {
	return " heard " + filename + " " + string(audio) + " ", nil
}

type fakePayments struct {
	available bool
	requests  []payments.CheckoutRequest
	event     *payments.PaymentEvent
}

func (p *fakePayments) Available() bool { return p.available }

func (p *fakePayments) CreateCheckout(_ context.Context, req payments.CheckoutRequest) (payments.Checkout, error) {
	p.requests = append(p.requests, req)
	return payments.Checkout{ProviderSessionID: "cks_dodo_1", URL: "https://checkout.dodopayments.com/test"}, nil
}

func (p *fakePayments) ParseWebhook(_ []byte, h payments.WebhookHeaders, _ time.Time) (*payments.PaymentEvent, error) {
	if h.Signature != "v1,good" {
		return nil, payments.ErrInvalidSignature
	}
	if p.event == nil {
		return nil, nil
	}
	event := *p.event
	event.WebhookID = h.ID
	return &event, nil
}

const tokenPrefix = "session:"

type fakeVerifier struct {
	available bool
}

func (v *fakeVerifier) Available() bool { return v.available }

func (v *fakeVerifier) Verify(_ context.Context, token string) (string, error) {
	clerkID, ok := strings.CutPrefix(token, tokenPrefix)
	if !ok || clerkID == "" {
		return "", errors.New("token not signed by clerk")
	}
	return clerkID, nil
}

type fakeDirectory struct {
	fail  bool
	calls int
}

func (d *fakeDirectory) PrimaryEmail(_ context.Context, clerkID string) (string, error) {
	d.calls++
	if d.fail {
		return "", errBoom
	}
	return clerkID + "@example.com", nil
}

type testEnv struct {
	t        *testing.T
	router   *gin.Engine
	store    *store.Memory
	auth     *fakeVerifier
	users    *fakeDirectory
	checks   *fakeModel
	rules    *fakeModel
	stt      *fakeSTT
	payments *fakePayments
	now      time.Time
}

func testConfig() *config.Config {
	open := config.Rate{PerMinute: 1000, Burst: 1000}
	return &config.Config{
		Env:        config.EnvDevelopment,
		WebsiteURL: "https://fuckalgo.app",
		Trial:      config.Trial{Voice: 2, Checks: 3},
		Limits:     config.Limits{BodyBytes: 1 << 20, TranscribeBodyBytes: 6 << 20, AudioBytes: 5 << 20},
		RateLimits: config.RateLimits{Checkout: open, VoiceToken: open, VoiceTranscribe: open, VoiceRules: open, Checks: open},
	}
}

func newEnv(t *testing.T) *testEnv {
	return newEnvWith(t, testConfig())
}

func newEnvWith(t *testing.T, cfg *config.Config) *testEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	e := &testEnv{
		t:        t,
		store:    store.NewMemory(),
		auth:     &fakeVerifier{available: true},
		users:    &fakeDirectory{},
		checks:   &fakeModel{available: true},
		rules:    &fakeModel{available: true},
		stt:      &fakeSTT{available: true},
		payments: &fakePayments{available: true},
		now:      time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC),
	}
	clock := func() time.Time { return e.now }
	router, err := httpapi.NewRouter(httpapi.Deps{
		Config:   cfg,
		Store:    e.store,
		Auth:     e.auth,
		Users:    e.users,
		STT:      e.stt,
		Checks:   checks.New(checks.NewModelJudge(e.checks), 24*time.Hour, 1000, clock),
		Voice:    voice.New(e.rules),
		Payments: e.payments,
		Now:      clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	e.router = router
	return e
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type reply struct {
	status int
	body   envelope
	header http.Header
}

func (r reply) code() string {
	if r.body.Error == nil {
		return ""
	}
	return r.body.Error.Code
}

func (r reply) into(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.body.Data, dst); err != nil {
		t.Fatalf("decoding data %s: %v", r.body.Data, err)
	}
}

func (e *testEnv) send(req *http.Request) reply {
	e.t.Helper()
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	var env envelope
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			e.t.Fatalf("response is not an envelope: %s", rec.Body.String())
		}
	}
	return reply{status: rec.Code, body: env, header: rec.Header()}
}

func (e *testEnv) do(method, path, token string, body any) reply {
	e.t.Helper()
	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reader = strings.NewReader(b)
	default:
		raw, _ := json.Marshal(b)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return e.send(req)
}

type meData struct {
	UserID  string `json:"userId"`
	Email   string `json:"email"`
	Licence struct {
		Status string `json:"status"`
		Trial  struct {
			VoiceLeft  int `json:"voiceLeft"`
			ChecksLeft int `json:"checksLeft"`
		} `json:"trial"`
	} `json:"licence"`
}

type account struct {
	ClerkID string
	Token   string
}

func (e *testEnv) signIn(clerkID string) account {
	return account{ClerkID: clerkID, Token: tokenPrefix + clerkID}
}

func (e *testEnv) me(token string) meData {
	e.t.Helper()
	r := e.do(http.MethodGet, "/v1/me", token, nil)
	if r.status != http.StatusOK {
		e.t.Fatalf("me: %d %s", r.status, r.code())
	}
	var data meData
	r.into(e.t, &data)
	return data
}

func (e *testEnv) webhookRequest(id, signature string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/dodo", strings.NewReader(`{"type":"payment.succeeded"}`))
	req.Header.Set("webhook-id", id)
	req.Header.Set("webhook-timestamp", "1")
	req.Header.Set("webhook-signature", signature)
	return req
}

func (e *testEnv) webhook(id string) reply {
	e.t.Helper()
	return e.send(e.webhookRequest(id, "v1,good"))
}

func (e *testEnv) pay(clerkID, sessionID string) {
	e.t.Helper()
	e.payments.event = &payments.PaymentEvent{EventType: payments.EventPaymentSucceeded, PaymentID: "pay_" + sessionID, CheckoutSessionID: sessionID, UserID: clerkID, Amount: 500, Currency: "USD"}
	if r := e.webhook("msg_" + sessionID); r.status != http.StatusOK {
		e.t.Fatalf("webhook failed: %d %s", r.status, r.code())
	}
}

func (e *testEnv) startCheckout(token string) string {
	e.t.Helper()
	r := e.do(http.MethodPost, "/v1/checkout", token, map[string]any{})
	if r.status != http.StatusOK {
		e.t.Fatalf("checkout failed: %d %s", r.status, r.code())
	}
	var data struct {
		SessionID string `json:"sessionId"`
	}
	r.into(e.t, &data)
	return data.SessionID
}

var errBoom = errors.New("boom")
