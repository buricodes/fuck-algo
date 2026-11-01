package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	r := e.do(http.MethodGet, "/healthz", "", nil)
	if r.status != http.StatusOK || string(r.body.Data) != `{"status":"ok","db":"ok"}` {
		t.Fatalf("healthz = %d %s", r.status, r.body.Data)
	}
}

func TestCreateInstallAndMe(t *testing.T) {
	e := newEnv(t)
	in := e.install()
	if !strings.HasPrefix(in.Token, "fa_inst_") || in.InstallID == "" {
		t.Fatalf("unexpected install %+v", in)
	}
	if in.Licence.Status != "trial" || in.Licence.KeyHint != nil || in.Licence.Trial.VoiceLeft != 2 || in.Licence.Trial.ChecksLeft != 3 {
		t.Fatalf("unexpected licence %+v", in.Licence)
	}
	me := e.me(in.Token)
	if me.InstallID != in.InstallID || me.Token != "" {
		t.Fatalf("me = %+v", me)
	}
}

func TestCreateInstallValidation(t *testing.T) {
	tests := []struct {
		name string
		body any
	}{
		{"not json", "{"},
		{"missing version", map[string]string{"browser": "chrome"}},
		{"missing browser", map[string]string{"version": "1.0.0"}},
		{"version too long", map[string]string{"version": strings.Repeat("1", 41), "browser": "chrome"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newEnv(t).do(http.MethodPost, "/v1/installs", "", tt.body)
			if r.status != http.StatusBadRequest || r.code() != "bad_request" {
				t.Fatalf("got %d %s", r.status, r.code())
			}
		})
	}
}

func TestAuthRequired(t *testing.T) {
	e := newEnv(t)
	for _, token := range []string{"", "nonsense", "fa_inst_unknown"} {
		r := e.do(http.MethodGet, "/v1/me", token, nil)
		if r.status != http.StatusUnauthorized || r.code() != "unauthorised" || r.body.OK {
			t.Fatalf("token %q: got %d %s", token, r.status, r.code())
		}
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
	big := `{"version":"` + strings.Repeat("a", 1<<20) + `","browser":"chrome"}`
	r := e.do(http.MethodPost, "/v1/installs", "", big)
	if r.status != http.StatusRequestEntityTooLarge || r.code() != "bad_request" {
		t.Fatalf("got %d %s", r.status, r.code())
	}
}

func TestRateLimitPerIP(t *testing.T) {
	cfg := testConfig()
	cfg.RateLimits.Installs.Burst = 2
	cfg.RateLimits.Installs.PerMinute = 1
	e := newEnvWith(t, cfg)
	body := map[string]string{"version": "1.0.0", "browser": "chrome"}
	for i, want := range []int{200, 200, 429} {
		r := e.do(http.MethodPost, "/v1/installs", "", body)
		if r.status != want {
			t.Fatalf("request %d: got %d %s, want %d", i, r.status, r.code(), want)
		}
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
