package checks

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/buricodes/fuck-algo/server/internal/domain"
	"github.com/buricodes/fuck-algo/server/internal/service/jev"
)

const (
	maxTopicChoices  = 80
	spoilerThreshold = 0.5
	noneChoice       = "none"
	spoilerQuestion  = "spoiler"
	topicQuestion    = "topic"
	spoilerPreamble  = "The person is avoiding spoilers for the titles listed in the question. They chose Paranoid mode: anything plausibly about a title counts, even when the title is never named (chapter or episode numbers, characters, places, leaks, reactions, theories, results)."
	spoilerAsk       = "Which watched title is this item plausibly about? Choose none only when it is clearly unrelated to every title."
	topicAsk         = "Which topic best describes what this item is about? Prefer the most specific topic that fits."
	noneDescription  = "Unrelated to all of these titles"
)

type Asker interface {
	Available() bool
	Ask(ctx context.Context, state string, questions map[string]jev.Question) (map[string]jev.Answer, error)
}

type JevJudge struct {
	asker Asker
}

func NewJevJudge(asker Asker) *JevJudge { return &JevJudge{asker: asker} }

func (j *JevJudge) Available() bool { return j.asker.Available() }

func (j *JevJudge) Judge(ctx context.Context, in Input) ([]Judgement, error) {
	topics := topicChoices(in.Topics)
	questions := buildQuestions(in.Paranoid, topics)
	out := make([]Judgement, len(in.Items))
	errs := make([]error, len(in.Items))
	var wg sync.WaitGroup
	for i, item := range in.Items {
		wg.Add(1)
		go func() {
			defer wg.Done()
			answers, err := j.asker.Ask(ctx, itemState(item, len(in.Paranoid) > 0), questions)
			if err != nil {
				errs[i] = err
				return
			}
			out[i] = Judgement{I: i, SpoilerOf: spoilerOf(answers[spoilerQuestion], in.Paranoid), Topic: topicOf(answers[topicQuestion], topics)}
		}()
	}
	wg.Wait()
	var judged []Judgement
	var firstErr error
	for i := range out {
		if errs[i] != nil {
			firstErr = firstOf(firstErr, errs[i])
			continue
		}
		judged = append(judged, out[i])
	}
	if len(judged) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return judged, nil
}

func buildQuestions(paranoid []domain.WatchContext, topics []string) map[string]jev.Question {
	q := map[string]jev.Question{
		topicQuestion: {Type: "choice", Instructions: topicAsk, Criteria: indexed("t", topics)},
	}
	if len(paranoid) > 0 {
		criteria := map[string]string{noneChoice: noneDescription}
		for i, w := range paranoid {
			criteria["w"+strconv.Itoa(i)] = describeWatch(w)
		}
		q[spoilerQuestion] = jev.Question{Type: "choice", Instructions: spoilerAsk, Criteria: criteria}
	}
	return q
}

func itemState(item domain.CheckItem, spoilers bool) string {
	var b strings.Builder
	if spoilers {
		b.WriteString(spoilerPreamble)
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "Feed item on %s.\nTitle: %s", item.Site, item.Title)
	if item.Channel != "" {
		fmt.Fprintf(&b, "\nChannel: %s", item.Channel)
	}
	if item.Text != "" {
		fmt.Fprintf(&b, "\nText: %s", item.Text)
	}
	return b.String()
}

func describeWatch(w domain.WatchContext) string {
	d := fmt.Sprintf("%s (%s)", w.Name, w.Type)
	if len(w.Keywords) > 0 {
		d += ". Related: " + strings.Join(w.Keywords, ", ")
	}
	return d
}

func spoilerOf(a jev.Answer, paranoid []domain.WatchContext) string {
	if a.Choice == "" || a.Choice == noneChoice || a.Probabilities[a.Choice] < spoilerThreshold {
		return ""
	}
	i, err := strconv.Atoi(strings.TrimPrefix(a.Choice, "w"))
	if err != nil || i < 0 || i >= len(paranoid) {
		return ""
	}
	return paranoid[i].Name
}

func topicOf(a jev.Answer, topics []string) string {
	i, err := strconv.Atoi(strings.TrimPrefix(a.Choice, "t"))
	if err != nil || i < 0 || i >= len(topics) {
		return ""
	}
	return topics[i]
}

func topicChoices(existing []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, maxTopicChoices)
	for _, t := range append(append([]string{}, existing...), baseTopics...) {
		t = strings.Join(strings.Fields(t), " ")
		key := strings.ToLower(t)
		if t == "" || seen[key] || strings.HasPrefix(key, "channel:") {
			continue
		}
		seen[key] = true
		out = append(out, t)
		if len(out) == maxTopicChoices {
			break
		}
	}
	return out
}

func indexed(prefix string, values []string) map[string]string {
	m := make(map[string]string, len(values))
	for i, v := range values {
		m[prefix+strconv.Itoa(i)] = v
	}
	return m
}

func firstOf(current, next error) error {
	if current != nil {
		return current
	}
	return next
}
