package voice

import (
	"encoding/json"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/buricodes/fuck-algo/server/internal/domain"
)

const (
	MaxRules           = 8
	maxNameChars       = 100
	maxTopicChars      = 60
	maxWordChars       = 60
	maxRuleKeywords    = 10
	maxRuleKeywordChar = 40
	maxExpiry          = 366 * 24 * time.Hour
)

var schema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "rules": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "kind": {"type": "string", "enum": ["spoiler", "repeat", "keyword"]},
          "name": {"type": "string"},
          "spoilerType": {"type": "string", "enum": ["anime", "game", "show", "movie", "sports", ""]},
          "level": {"type": "integer"},
          "keywords": {"type": "array", "items": {"type": "string"}},
          "topic": {"type": "string"},
          "sites": {"type": "array", "items": {"type": "string", "enum": ["youtube", "x", "instagram", "web"]}},
          "word": {"type": "string"},
          "action": {"type": "string", "enum": ["hide", "blur", "skip", ""]},
          "expiresAt": {"type": "string"}
        },
        "required": ["kind", "name", "spoilerType", "level", "keywords", "topic", "sites", "word", "action", "expiresAt"],
        "additionalProperties": false
      }
    },
    "summary": {"type": "string"}
  },
  "required": ["rules", "summary"],
  "additionalProperties": false
}`)

type modelRule struct {
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	SpoilerType string   `json:"spoilerType"`
	Level       int      `json:"level"`
	Keywords    []string `json:"keywords"`
	Topic       string   `json:"topic"`
	Sites       []string `json:"sites"`
	Word        string   `json:"word"`
	Action      string   `json:"action"`
	ExpiresAt   string   `json:"expiresAt"`
}

type modelOutput struct {
	Rules   []modelRule `json:"rules"`
	Summary string      `json:"summary"`
}

func toRules(in []modelRule, now time.Time) []domain.ProposedRule {
	rules := []domain.ProposedRule{}
	for _, r := range in {
		if rule, ok := toRule(r, now); ok && len(rules) < MaxRules {
			rules = append(rules, rule)
		}
	}
	return rules
}

func toRule(r modelRule, now time.Time) (domain.ProposedRule, bool) {
	switch r.Kind {
	case "spoiler":
		return spoilerRule(r)
	case "repeat":
		return repeatRule(r)
	case "keyword":
		return keywordRule(r, now)
	}
	return nil, false
}

func spoilerRule(r modelRule) (domain.ProposedRule, bool) {
	name := clean(r.Name)
	kind := domain.SpoilerType(r.SpoilerType)
	if !fits(name, maxNameChars) || !kind.Valid() {
		return nil, false
	}
	level := domain.Strictness(min(max(r.Level, 0), 2))
	return domain.SpoilerRule{Kind: "spoiler", Name: name, Type: kind, Level: level, Keywords: keywords(r.Keywords, name)}, true
}

func repeatRule(r modelRule) (domain.ProposedRule, bool) {
	topic := clean(r.Topic)
	if !fits(topic, maxTopicChars) {
		return nil, false
	}
	return domain.RepeatRule{Kind: "repeat", Topic: topic, Sites: sites(r.Sites)}, true
}

func keywordRule(r modelRule, now time.Time) (domain.ProposedRule, bool) {
	word := clean(r.Word)
	action := domain.KeywordAction(r.Action)
	if !fits(word, maxWordChars) {
		return nil, false
	}
	if !action.Valid() {
		action = "hide"
	}
	expiresAt, ok := expiry(r.ExpiresAt, now)
	if !ok {
		return nil, false
	}
	return domain.KeywordRule{Kind: "keyword", Word: word, Action: action, ExpiresAt: expiresAt}, true
}

func expiry(raw string, now time.Time) (*string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, true
	}
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil || !at.After(now) || at.Sub(now) > maxExpiry {
		return nil, false
	}
	formatted := at.UTC().Format(time.RFC3339)
	return &formatted, true
}

func keywords(in []string, title string) []string {
	out := []string{}
	for _, k := range in {
		k = strings.ToLower(clean(k))
		if fits(k, maxRuleKeywordChar) && !strings.EqualFold(k, title) && !slices.Contains(out, k) && len(out) < maxRuleKeywords {
			out = append(out, k)
		}
	}
	return out
}

func sites(in []string) []domain.Site {
	var out []domain.Site
	for _, s := range in {
		site := domain.Site(s)
		if site.Valid() && !slices.Contains(out, site) {
			out = append(out, site)
		}
	}
	if len(out) == 0 {
		return slices.Clone(domain.Sites)
	}
	return out
}

func clean(s string) string { return strings.Join(strings.Fields(s), " ") }

func fits(s string, maxChars int) bool {
	return s != "" && utf8.RuneCountInString(s) <= maxChars
}
