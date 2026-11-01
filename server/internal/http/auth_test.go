package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/buricodes/fuck-algo/server/internal/config"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	r := e.do(http.MethodGet, "/healthz", "", nil)
	if r.status != http.StatusOK || string(r.body.Data) != `{"status":"ok","db":"ok"}` {
		t.Fatalf("healthz = %d %s", r.status, r.body.Data)
	}
}

func TestMeCreatesUserOnce(t *testing.T) {
	e := newEnv(t)
	u := e.signIn("user_a")
	me := e.me(u.Token)
	if me.UserID != "user_a" || me.Email != "user_a@example.com" {
		t.Fatalf("me = %+v", me)
	}
	if me.Licence.Status != "trial" || me.Licence.Trial.VoiceLeft != 2 || me.Licence.Trial.ChecksLeft != 3 {
		t.Fatalf("unexpected licence %+v", me.Licence)
	}
	e.me(u.Token)
	if e.users.calls != 1 {
		t.Fatalf("clerk email lookups = %d, want 1", e.users.calls)
	}
	if other := e.me(e.signIn("user_b").Token); other.UserID != "user_b" || other.Licence.Trial.VoiceLeft != 2 {
		t.Fatalf("second user = %+v", other)
	}
}

func TestMeRetriesEmailLookup(t *testing.T) {
	e := newEnv(t)
	u := e.signIn("user_a")
	e.users.fail = true
	if me := e.me(u.Token); me.UserID != "user_a" || me.Email != "" {
		t.Fatalf("me without email = %+v", me)
	}
	e.users.fail = false
	if me := e.me(u.Token); me.Email != "user_a@example.com" {
		t.Fatalf("email not filled in on a later call: %+v", me)
	}
	calls := e.users.calls
	e.me(u.Token)
	if e.users.calls != calls {
		t.Fatal("email looked up again after it was stored")
	}
}

func TestAuthRequired(t *testing.T) {
	e := newEnv(t)
	routes := []struct{ method, path string }{
		{http.MethodGet, "/v1/me"},
		{http.MethodPost, "/v1/checkout"},
		{http.MethodGet, "/v1/checkout/cks_x"},
		{http.MethodPost, "/v1/voice/token"},
		{http.MethodPost, "/v1/voice/transcribe"},
		{http.MethodPost, "/v1/voice/rules"},
		{http.MethodPost, "/v1/checks"},
	}
	for _, route := range routes {
		for _, token := range []string{"", "nonsense", tokenPrefix} {
			r := e.do(route.method, route.path, token, "{}")
			if r.status != http.StatusUnauthorized || r.code() != "unauthorised" || r.body.OK {
				t.Fatalf("%s %s with %q: got %d %s", route.method, route.path, token, r.status, r.code())
			}
		}
	}
	if e.users.calls != 0 {
		t.Fatal("rejected tokens must not reach clerk")
	}
}

func TestAuthUnavailableWithoutClerk(t *testing.T) {
	e := newEnv(t)
	e.auth.available = false
	r := e.do(http.MethodGet, "/v1/me", e.signIn("user_a").Token, nil)
	if r.status != http.StatusServiceUnavailable || r.code() != "unavailable" {
		t.Fatalf("got %d %s", r.status, r.code())
	}
	if r := e.do(http.MethodGet, "/healthz", "", nil); r.status != http.StatusOK {
		t.Fatalf("healthz needs no auth: %d", r.status)
	}
}

func TestUnknownRoute(t *testing.T) {
	r := newEnv(t).do(http.MethodGet, "/v1/nope", "", nil)
	if r.status != http.StatusNotFound || r.code() != "not_found" {
		t.Fatalf("got %d %s", r.status, r.code())
	}
}

func TestBodyLimit(t *testing.T) {
	e := newEnv(t)
	big := `{"transcript":"` + strings.Repeat("a", 1<<20) + `","now":"2026-11-01T10:00:00Z"}`
	r := e.do(http.MethodPost, "/v1/voice/rules", e.signIn("user_a").Token, big)
	if r.status != http.StatusRequestEntityTooLarge || r.code() != "bad_request" {
		t.Fatalf("got %d %s", r.status, r.code())
	}
}

func TestRateLimitPerUser(t *testing.T) {
	cfg := testConfig()
	cfg.RateLimits.VoiceToken = config.Rate{PerMinute: 1, Burst: 2}
	cfg.Trial.Voice = 10
	e := newEnvWith(t, cfg)
	a, b := e.signIn("user_a"), e.signIn("user_b")
	for i, want := range []int{200, 200, 429} {
		if r := e.do(http.MethodPost, "/v1/voice/token", a.Token, nil); r.status != want {
			t.Fatalf("request %d: got %d %s, want %d", i, r.status, r.code(), want)
		}
	}
	if r := e.do(http.MethodPost, "/v1/voice/token", b.Token, nil); r.status != http.StatusOK {
		t.Fatalf("another user was limited: %d %s", r.status, r.code())
	}
}

func TestCORS(t *testing.T) {
	cfg := testConfig()
	tests := []struct {
		name    string
		origins []string
		env     string
		origin  string
		allowed bool
	}{
		{"website", nil, "development", "https://fuckalgo.app", true},
		{"any extension in development", nil, "development", "chrome-extension://abc", true},
		{"any extension refused in production", nil, "production", "chrome-extension://abc", false},
		{"listed extension", []string{"chrome-extension://abc"}, "production", "chrome-extension://abc", true},
		{"unlisted extension", []string{"chrome-extension://abc"}, "development", "chrome-extension://xyz", false},
		{"other site", nil, "development", "https://evil.example", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := *cfg
			c.ExtensionOrigins, c.Env = tt.origins, tt.env
			e := newEnvWith(t, &c)
			req := httptest.NewRequest(http.MethodOptions, "/v1/checks", nil)
			req.Header.Set("Origin", tt.origin)
			req.Header.Set("Access-Control-Request-Method", "POST")
			r := e.send(req)
			got := r.header.Get("Access-Control-Allow-Origin") == tt.origin
			if r.status != http.StatusNoContent || got != tt.allowed {
				t.Fatalf("status %d, allow origin %q, want allowed %v", r.status, r.header.Get("Access-Control-Allow-Origin"), tt.allowed)
			}
		})
	}
}
