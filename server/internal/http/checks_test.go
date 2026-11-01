package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type modelItem struct {
	I     int    `json:"i"`
	Title string `json:"title"`
}

func echoChecks(user string) (string, error) {
	var in struct {
		Items []modelItem `json:"items"`
	}
	if err := json.Unmarshal([]byte(user), &in); err != nil {
		return "", err
	}
	var results []string
	for _, item := range in.Items {
		spoiler := ""
		if strings.Contains(item.Title, "Ch.") {
			spoiler = "black clover"
		}
		results = append(results, fmt.Sprintf(`{"i":%d,"spoilerOf":%q,"topic":"  Manga   Reactions "}`, item.I, spoiler))
	}
	return `{"results":[` + strings.Join(results, ",") + `]}`, nil
}

func checkItems(ids ...string) []map[string]string {
	var items []map[string]string
	for _, id := range ids {
		items = append(items, map[string]string{"id": id, "site": "youtube", "title": "Ch. 371 just changed everything " + id, "channel": "MangaTalk"})
	}
	return items
}

type checkResult struct {
	ID        string  `json:"id"`
	SpoilerOf *string `json:"spoilerOf"`
	Topic     string  `json:"topic"`
}

func runChecks(t *testing.T, e *testEnv, token string, level int, ids ...string) ([]checkResult, reply) {
	t.Helper()
	r := e.do(http.MethodPost, "/v1/checks", token, map[string]any{
		"items":     checkItems(ids...),
		"watchlist": []map[string]any{{"name": "Black Clover", "type": "anime", "level": level, "keywords": []string{"Asta"}}, {"name": "Frieren", "type": "anime", "level": 1, "keywords": []string{}}},
		"topics":    []string{"Gym bro advice"},
	})
	var data struct {
		Results []checkResult `json:"results"`
	}
	if r.status == http.StatusOK {
		r.into(t, &data)
	}
	return data.Results, r
}

func TestChecksJudgesCachesAndSpends(t *testing.T) {
	e := newEnv(t)
	in := e.install()
	e.checks.reply = echoChecks

	results, r := runChecks(t, e, in.Token, 2, "yt:a", "yt:b")
	if r.status != http.StatusOK || len(results) != 2 || e.checks.calls != 1 {
		t.Fatalf("first call: %d %s, %d results, %d model calls", r.status, r.code(), len(results), e.checks.calls)
	}
	if results[0].ID != "yt:a" || results[0].SpoilerOf == nil || *results[0].SpoilerOf != "Black Clover" || results[0].Topic != "Manga Reactions" {
		t.Fatalf("unexpected result %+v", results[0])
	}
	if strings.Contains(e.checks.lastUser, "Frieren") || !strings.Contains(e.checks.lastUser, "Gym bro advice") {
		t.Fatalf("model input should hold only Paranoid titles and the existing topics: %s", e.checks.lastUser)
	}

	results, _ = runChecks(t, e, in.Token, 2, "yt:b", "yt:a")
	if len(results) != 2 || results[0].ID != "yt:b" || e.checks.calls != 1 {
		t.Fatalf("cached call: %d results, %d model calls", len(results), e.checks.calls)
	}
	if left := e.me(in.Token).Licence.Trial.ChecksLeft; left != 1 {
		t.Fatalf("checksLeft = %d, want 1", left)
	}

	results, _ = runChecks(t, e, in.Token, 2, "yt:a", "yt:c", "yt:d")
	if len(results) != 2 || results[1].ID != "yt:c" {
		t.Fatalf("partial trial: got %+v, want yt:a cached and yt:c judged", results)
	}
	if _, r = runChecks(t, e, in.Token, 2, "yt:e"); r.status != http.StatusPaymentRequired || r.code() != "trial_exhausted" {
		t.Fatalf("exhausted: got %d %s", r.status, r.code())
	}
}

func TestChecksWatchlistChangesCacheKey(t *testing.T) {
	e := newEnv(t)
	in := e.install()
	e.checks.reply = echoChecks
	runChecks(t, e, in.Token, 2, "yt:a")
	results, _ := runChecks(t, e, in.Token, 1, "yt:a")
	if e.checks.calls != 2 || results[0].SpoilerOf != nil {
		t.Fatalf("calls = %d, result %+v: a Balanced title must not be judged and must miss the cache", e.checks.calls, results[0])
	}
}

func TestChecksRefundsOnFailure(t *testing.T) {
	e := newEnv(t)
	in := e.install()
	e.checks.reply = func(string) (string, error) { return `{"results":[{"i":0,"spoilerOf":"","topic":"General"}]}`, nil }
	results, _ := runChecks(t, e, in.Token, 2, "yt:a", "yt:b")
	if len(results) != 1 || e.me(in.Token).Licence.Trial.ChecksLeft != 2 {
		t.Fatalf("item the model skipped should be refunded")
	}
	e.checks.reply = func(string) (string, error) { return "", errBoom }
	runChecks(t, e, in.Token, 2, "yt:c")
	if left := e.me(in.Token).Licence.Trial.ChecksLeft; left != 2 {
		t.Fatalf("checksLeft = %d after a failed call, want 2", left)
	}
}

func TestChecksValidation(t *testing.T) {
	long := strings.Repeat("a", 301)
	var many []string
	for i := range 26 {
		many = append(many, fmt.Sprint(i))
	}
	tests := []struct {
		name string
		body map[string]any
	}{
		{"no items", map[string]any{"items": []any{}}},
		{"too many items", map[string]any{"items": checkItems(many...)}},
		{"duplicate ids", map[string]any{"items": checkItems("a", "a")}},
		{"bad site", map[string]any{"items": []map[string]string{{"id": "a", "site": "tiktok", "title": "x"}}}},
		{"title too long", map[string]any{"items": []map[string]string{{"id": "a", "site": "web", "title": long}}}},
		{"text too long", map[string]any{"items": []map[string]string{{"id": "a", "site": "web", "title": "x", "text": strings.Repeat("a", 601)}}}},
		{"bad level", map[string]any{"items": checkItems("a"), "watchlist": []map[string]any{{"name": "X", "type": "anime", "level": 3}}}},
		{"bad type", map[string]any{"items": checkItems("a"), "watchlist": []map[string]any{{"name": "X", "type": "book", "level": 2}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			in := e.install()
			r := e.do(http.MethodPost, "/v1/checks", in.Token, tt.body)
			if r.status != http.StatusBadRequest || r.code() != "bad_request" {
				t.Fatalf("got %d %s", r.status, r.code())
			}
		})
	}
}

func TestChecksUnavailable(t *testing.T) {
	e := newEnv(t)
	in := e.install()
	e.checks.available = false
	if _, r := runChecks(t, e, in.Token, 2, "a"); r.status != http.StatusServiceUnavailable || r.code() != "unavailable" {
		t.Fatalf("got %d %s", r.status, r.code())
	}
}
