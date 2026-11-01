package httpapi_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
)

func TestVoiceTokenSpendsTrial(t *testing.T) {
	e := newEnv(t)
	in := e.install()
	for i, want := range []int{200, 200, 402} {
		r := e.do(http.MethodPost, "/v1/voice/token", in.Token, nil)
		if r.status != want {
			t.Fatalf("call %d: got %d %s, want %d", i, r.status, r.code(), want)
		}
		if want == 402 && r.code() != "trial_exhausted" {
			t.Fatalf("code = %s, want trial_exhausted", r.code())
		}
	}
	if left := e.me(in.Token).Licence.Trial.VoiceLeft; left != 0 {
		t.Fatalf("voiceLeft = %d, want 0", left)
	}
}

func TestVoiceTokenRefundsOnUpstreamFailure(t *testing.T) {
	e := newEnv(t)
	in := e.install()
	e.stt.fail = true
	r := e.do(http.MethodPost, "/v1/voice/token", in.Token, nil)
	if r.status != http.StatusBadGateway || r.code() != "upstream" {
		t.Fatalf("got %d %s", r.status, r.code())
	}
	if left := e.me(in.Token).Licence.Trial.VoiceLeft; left != 2 {
		t.Fatalf("voiceLeft = %d, want the use refunded", left)
	}
}

func TestVoiceUnavailable(t *testing.T) {
	e := newEnv(t)
	in := e.install()
	e.stt.available = false
	e.rules.available = false
	for _, path := range []string{"/v1/voice/token", "/v1/voice/transcribe", "/v1/voice/rules"} {
		r := e.do(http.MethodPost, path, in.Token, "{}")
		if r.status != http.StatusServiceUnavailable || r.code() != "unavailable" {
			t.Fatalf("%s: got %d %s", path, r.status, r.code())
		}
	}
	if left := e.me(in.Token).Licence.Trial.VoiceLeft; left != 2 {
		t.Fatalf("voiceLeft = %d, want nothing spent", left)
	}
}

func audioRequest(t *testing.T, token, contentType, filename string, audio []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="audio"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)
	part, _ := w.CreatePart(h)
	_, _ = part.Write(audio)
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/v1/voice/transcribe", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestVoiceTranscribe(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		filename    string
		audio       []byte
		status      int
		transcript  string
	}{
		{"webm with codecs", "audio/webm;codecs=opus", "clip", []byte("abc"), 200, "heard audio.webm abc"},
		{"ogg", "audio/ogg", "clip.ogg", []byte("abc"), 200, "heard audio.ogg abc"},
		{"octet stream with webm name", "application/octet-stream", "clip.webm", []byte("abc"), 200, "heard audio.webm abc"},
		{"wav refused", "audio/wav", "clip.wav", []byte("abc"), 400, ""},
		{"empty refused", "audio/webm", "clip.webm", nil, 400, ""},
		{"over 5 MB refused", "audio/webm", "clip.webm", bytes.Repeat([]byte("a"), 5<<20+1), 400, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			in := e.install()
			r := e.send(audioRequest(t, in.Token, tt.contentType, tt.filename, tt.audio))
			if r.status != tt.status {
				t.Fatalf("got %d %s, want %d", r.status, r.code(), tt.status)
			}
			if tt.status != 200 {
				return
			}
			var data struct{ Transcript string }
			r.into(t, &data)
			if data.Transcript != tt.transcript {
				t.Fatalf("transcript = %q, want %q", data.Transcript, tt.transcript)
			}
			if left := e.me(in.Token).Licence.Trial.VoiceLeft; left != 2 {
				t.Fatalf("transcribe spent a voice use")
			}
		})
	}
}

const designRulesReply = `{
  "rules": [
    {"kind": "spoiler", "name": " GTA 6 ", "spoilerType": "game", "level": 2, "keywords": ["Leak", "trailer", "ending", "leak", "GTA 6"], "topic": "", "sites": [], "word": "", "action": "", "expiresAt": ""},
    {"kind": "repeat", "name": "", "spoilerType": "", "level": 0, "keywords": [], "topic": "Gym bro advice", "sites": ["youtube", "instagram", "youtube"], "word": "", "action": "", "expiresAt": ""},
    {"kind": "keyword", "name": "", "spoilerType": "", "level": 0, "keywords": [], "topic": "", "sites": [], "word": "crypto", "action": "blur", "expiresAt": ""},
    {"kind": "keyword", "name": "", "spoilerType": "", "level": 0, "keywords": [], "topic": "", "sites": [], "word": "election", "action": "blur", "expiresAt": "2026-11-06T23:59:00Z"},
    {"kind": "keyword", "name": "", "spoilerType": "", "level": 0, "keywords": [], "topic": "", "sites": [], "word": "stale", "action": "hide", "expiresAt": "2020-01-01T00:00:00Z"},
    {"kind": "spoiler", "name": "Nameless", "spoilerType": "book", "level": 2, "keywords": [], "topic": "", "sites": [], "word": "", "action": "", "expiresAt": ""},
    {"kind": "repeat", "name": "", "spoilerType": "", "level": 0, "keywords": [], "topic": "Everywhere", "sites": [], "word": "", "action": "", "expiresAt": ""}
  ],
  "summary": "Three rules: one spoiler title, one topic, one keyword."
}`

func TestVoiceRules(t *testing.T) {
	e := newEnv(t)
	in := e.install()
	e.rules.reply = func(string) (string, error) { return designRulesReply, nil }
	r := e.do(http.MethodPost, "/v1/voice/rules", in.Token, map[string]any{
		"transcript": "No GTA 6 spoilers, I'm waiting for the PC release. Way fewer gym bro videos. And blur anything about crypto.",
		"now":        "2026-11-01T10:00:00Z",
		"context":    map[string]any{"watchlist": []string{"Black Clover"}, "keywords": []string{"MrBeast"}, "topics": []string{"Gym bro advice"}},
	})
	if r.status != http.StatusOK {
		t.Fatalf("got %d %s", r.status, r.code())
	}
	var data struct {
		Rules   []json.RawMessage `json:"rules"`
		Summary string            `json:"summary"`
	}
	r.into(t, &data)
	want := []string{
		`{"kind":"spoiler","name":"GTA 6","type":"game","level":2,"keywords":["leak","trailer","ending"]}`,
		`{"kind":"repeat","topic":"Gym bro advice","sites":["youtube","instagram"]}`,
		`{"kind":"keyword","word":"crypto","action":"blur","expiresAt":null}`,
		`{"kind":"keyword","word":"election","action":"blur","expiresAt":"2026-11-06T23:59:00Z"}`,
		`{"kind":"repeat","topic":"Everywhere","sites":["youtube","x","instagram","web"]}`,
	}
	if len(data.Rules) != len(want) {
		t.Fatalf("got %d rules, want %d: %s", len(data.Rules), len(want), r.body.Data)
	}
	for i := range want {
		if string(data.Rules[i]) != want[i] {
			t.Fatalf("rule %d = %s, want %s", i, data.Rules[i], want[i])
		}
	}
	if !strings.Contains(e.rules.lastUser, "Sunday") || !strings.Contains(e.rules.lastUser, `"topics":["Gym bro advice"]`) {
		t.Fatalf("model input is missing the weekday or context: %s", e.rules.lastUser)
	}
	if left := e.me(in.Token).Licence.Trial.VoiceLeft; left != 2 {
		t.Fatalf("rules spent a voice use")
	}
}

func TestVoiceRulesEmpty(t *testing.T) {
	e := newEnv(t)
	in := e.install()
	e.rules.reply = func(string) (string, error) { return `{"rules": [], "summary": ""}`, nil }
	r := e.do(http.MethodPost, "/v1/voice/rules", in.Token, map[string]any{"transcript": "hmm", "now": "2026-11-01T10:00:00Z"})
	if r.status != http.StatusOK || string(r.body.Data) != `{"rules":[],"summary":"Nothing to add: I didn't catch a rule in that."}` {
		t.Fatalf("got %d %s", r.status, r.body.Data)
	}
}

func TestVoiceRulesValidation(t *testing.T) {
	tests := []struct {
		name string
		body map[string]any
	}{
		{"empty transcript", map[string]any{"transcript": "  ", "now": "2026-11-01T10:00:00Z"}},
		{"transcript too long", map[string]any{"transcript": strings.Repeat("a", 2001), "now": "2026-11-01T10:00:00Z"}},
		{"bad now", map[string]any{"transcript": "hi", "now": "tomorrow"}},
		{"context entry too long", map[string]any{"transcript": "hi", "now": "2026-11-01T10:00:00Z", "context": map[string]any{"topics": []string{strings.Repeat("a", 101)}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			in := e.install()
			r := e.do(http.MethodPost, "/v1/voice/rules", in.Token, tt.body)
			if r.status != http.StatusBadRequest || r.code() != "bad_request" || e.rules.calls != 0 {
				t.Fatalf("got %d %s after %d model calls", r.status, r.code(), e.rules.calls)
			}
		})
	}
}

func TestVoiceRulesUpstreamError(t *testing.T) {
	e := newEnv(t)
	in := e.install()
	e.rules.reply = func(string) (string, error) { return "", errBoom }
	r := e.do(http.MethodPost, "/v1/voice/rules", in.Token, map[string]any{"transcript": "hi", "now": "2026-11-01T10:00:00Z"})
	if r.status != http.StatusInternalServerError || r.code() != "internal" {
		t.Fatalf("got %d %s", r.status, r.code())
	}
}
