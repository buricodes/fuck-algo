package voice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/buricodes/fuck-algo/server/internal/domain"
	"github.com/buricodes/fuck-algo/server/internal/prompts"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
)

const (
	MaxTranscriptChars = 2000
	maxContextEntries  = 200
	maxContextChars    = 100
)

type Model interface {
	Available() bool
	JSON(ctx context.Context, system, user string, schema json.RawMessage) ([]byte, error)
}

type Context struct {
	Watchlist []string `json:"watchlist"`
	Keywords  []string `json:"keywords"`
	Topics    []string `json:"topics"`
}

type Request struct {
	Transcript string  `json:"transcript"`
	Now        string  `json:"now"`
	Context    Context `json:"context"`
}

type Result struct {
	Rules   []domain.ProposedRule `json:"rules"`
	Summary string                `json:"summary"`
}

type Service struct {
	model Model
}

func New(model Model) *Service { return &Service{model: model} }

func (s *Service) Available() bool { return s.model.Available() }

func Validate(req Request) (time.Time, error) {
	transcript := strings.TrimSpace(req.Transcript)
	if transcript == "" || utf8.RuneCountInString(transcript) > MaxTranscriptChars {
		return time.Time{}, fmt.Errorf("transcript must be 1 to %d characters", MaxTranscriptChars)
	}
	now, err := time.Parse(time.RFC3339, req.Now)
	if err != nil {
		return time.Time{}, errors.New("now must be an RFC 3339 timestamp")
	}
	for name, list := range map[string][]string{"watchlist": req.Context.Watchlist, "keywords": req.Context.Keywords, "topics": req.Context.Topics} {
		if err := validateList(name, list); err != nil {
			return time.Time{}, err
		}
	}
	return now.UTC(), nil
}

func (s *Service) Rules(ctx context.Context, req Request, now time.Time) (Result, error) {
	raw, err := s.model.JSON(ctx, prompts.VoiceRules, userMessage(req, now), schema)
	if err != nil {
		return Result{}, err
	}
	var out modelOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return Result{}, &apperr.UpstreamError{Provider: "anthropic", Err: fmt.Errorf("voice rules: decoding reply: %w", err)}
	}
	rules := toRules(out.Rules, now)
	return Result{Rules: rules, Summary: summary(out.Summary, len(rules))}, nil
}

func userMessage(req Request, now time.Time) string {
	setup, _ := json.Marshal(map[string][]string{
		"watchlist": orEmpty(req.Context.Watchlist),
		"keywords":  orEmpty(req.Context.Keywords),
		"topics":    orEmpty(req.Context.Topics),
	})
	return fmt.Sprintf("Current time: %s (%s)\nExisting setup: %s\n\n<transcript>\n%s\n</transcript>",
		now.Format(time.RFC3339), now.Weekday(), setup, strings.TrimSpace(req.Transcript))
}

func validateList(name string, list []string) error {
	if len(list) > maxContextEntries {
		return fmt.Errorf("context.%s may hold at most %d entries", name, maxContextEntries)
	}
	for _, v := range list {
		if utf8.RuneCountInString(v) > maxContextChars {
			return fmt.Errorf("each of context.%s may be at most %d characters", name, maxContextChars)
		}
	}
	return nil
}

func summary(text string, count int) string {
	text = strings.TrimSpace(text)
	switch {
	case count == 0:
		return "Nothing to add: I didn't catch a rule in that."
	case text != "" && utf8.RuneCountInString(text) <= 300:
		return text
	case count == 1:
		return "One rule."
	}
	return fmt.Sprintf("%d rules.", count)
}

func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
