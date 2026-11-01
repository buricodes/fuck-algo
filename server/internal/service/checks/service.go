package checks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"time"

	"github.com/buricodes/fuck-algo/server/internal/domain"
)

const (
	maxTopicChars = 60
	fallbackTopic = "General"
)

type Judgement struct {
	I         int
	SpoilerOf string
	Topic     string
}

type Input struct {
	Paranoid []domain.WatchContext
	Topics   []string
	Items    []domain.CheckItem
}

type Judge interface {
	Available() bool
	Judge(ctx context.Context, in Input) ([]Judgement, error)
}

type Request struct {
	Items     []domain.CheckItem    `json:"items"`
	Watchlist []domain.WatchContext `json:"watchlist"`
	Topics    []string              `json:"topics"`
}

type Plan struct {
	Results   map[string]domain.CheckResult
	Pending   []domain.CheckItem
	paranoid  []domain.WatchContext
	topics    []string
	watchHash string
}

type Service struct {
	judge Judge
	cache *cache
	now   func() time.Time
}

func New(judge Judge, ttl time.Duration, size int, now func() time.Time) *Service {
	return &Service{judge: judge, cache: newCache(ttl, size), now: now}
}

func (s *Service) Available() bool { return s.judge.Available() }

func (s *Service) Plan(req Request) *Plan {
	paranoid := paranoidTitles(req.Watchlist)
	p := &Plan{
		Results:   map[string]domain.CheckResult{},
		paranoid:  paranoid,
		topics:    req.Topics,
		watchHash: watchlistHash(paranoid),
	}
	now := s.now()
	for _, item := range req.Items {
		if v, ok := s.cache.get(p.cacheKey(item.ID), now); ok {
			p.Results[item.ID] = toResult(item.ID, v)
			continue
		}
		p.Pending = append(p.Pending, item)
	}
	return p
}

func (s *Service) Judge(ctx context.Context, p *Plan, items []domain.CheckItem) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	out, err := s.judge.Judge(ctx, Input{Paranoid: p.paranoid, Topics: p.topics, Items: items})
	if err != nil {
		return 0, err
	}
	return s.apply(p, items, out), nil
}

func (s *Service) apply(p *Plan, items []domain.CheckItem, out []Judgement) int {
	now := s.now()
	judged := 0
	for _, r := range out {
		if r.I < 0 || r.I >= len(items) {
			continue
		}
		item := items[r.I]
		if _, done := p.Results[item.ID]; done {
			continue
		}
		v := verdict{spoilerOf: matchTitle(p.paranoid, r.SpoilerOf), topic: cleanTopic(r.Topic)}
		s.cache.put(p.cacheKey(item.ID), v, now)
		p.Results[item.ID] = toResult(item.ID, v)
		judged++
	}
	return judged
}

func (p *Plan) cacheKey(itemID string) string { return p.watchHash + "|" + itemID }

func paranoidTitles(watchlist []domain.WatchContext) []domain.WatchContext {
	var out []domain.WatchContext
	for _, w := range watchlist {
		if w.Level == domain.Paranoid {
			out = append(out, w)
		}
	}
	return out
}

func watchlistHash(paranoid []domain.WatchContext) string {
	parts := make([]string, 0, len(paranoid))
	for _, w := range paranoid {
		keywords := slices.Clone(w.Keywords)
		for i := range keywords {
			keywords[i] = strings.ToLower(strings.TrimSpace(keywords[i]))
		}
		slices.Sort(keywords)
		parts = append(parts, strings.ToLower(strings.TrimSpace(w.Name))+"\x1f"+string(w.Type)+"\x1f"+strings.Join(keywords, "\x1e"))
	}
	slices.Sort(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1d")))
	return hex.EncodeToString(sum[:16])
}

func matchTitle(paranoid []domain.WatchContext, name string) string {
	name = strings.TrimSpace(name)
	for _, w := range paranoid {
		if name != "" && strings.EqualFold(w.Name, name) {
			return w.Name
		}
	}
	return ""
}

func cleanTopic(topic string) string {
	topic = strings.Join(strings.Fields(topic), " ")
	if topic == "" {
		return fallbackTopic
	}
	if r := []rune(topic); len(r) > maxTopicChars {
		return strings.TrimSpace(string(r[:maxTopicChars]))
	}
	return topic
}

func toResult(id string, v verdict) domain.CheckResult {
	r := domain.CheckResult{ID: id, Topic: v.topic}
	if v.spoilerOf != "" {
		title := v.spoilerOf
		r.SpoilerOf = &title
	}
	return r
}
