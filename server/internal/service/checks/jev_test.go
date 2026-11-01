package checks

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/buricodes/fuck-algo/server/internal/domain"
	"github.com/buricodes/fuck-algo/server/internal/service/jev"
)

type fakeAsker struct {
	answer func(state string, q map[string]jev.Question) (map[string]jev.Answer, error)
}

func (f fakeAsker) Available() bool { return true }

func (f fakeAsker) Ask(_ context.Context, state string, q map[string]jev.Question) (map[string]jev.Answer, error) {
	return f.answer(state, q)
}

func choiceOf(q jev.Question, label string) string {
	for id, v := range q.Criteria {
		if v == label {
			return id
		}
	}
	return ""
}

func TestJevJudge(t *testing.T) {
	paranoid := []domain.WatchContext{{Name: "Black Clover", Type: domain.SpoilerAnime, Level: domain.Paranoid, Keywords: []string{"Asta"}}}
	asker := fakeAsker{answer: func(state string, q map[string]jev.Question) (map[string]jev.Answer, error) {
		if strings.Contains(state, "broken") {
			return nil, errors.New("boom")
		}
		spoiler := jev.Answer{Choice: "none", Probabilities: map[string]float64{"none": 0.9}}
		if strings.Contains(state, "Ch. 371") {
			spoiler = jev.Answer{Choice: "w0", Probabilities: map[string]float64{"w0": 0.7, "none": 0.3}}
		}
		if strings.Contains(state, "Asta?") {
			spoiler = jev.Answer{Choice: "w0", Probabilities: map[string]float64{"w0": 0.4, "none": 0.6}}
		}
		topic := choiceOf(q[topicQuestion], "Anime and manga")
		return map[string]jev.Answer{
			spoilerQuestion: spoiler,
			topicQuestion:   {Choice: topic, Probabilities: map[string]float64{topic: 1}},
		}, nil
	}}

	items := []domain.CheckItem{
		{ID: "a", Site: "youtube", Title: "Ch. 371 just changed everything"},
		{ID: "b", Site: "youtube", Title: "Asta? maybe"},
		{ID: "c", Site: "youtube", Title: "broken item"},
	}
	got, err := NewJevJudge(asker).Judge(context.Background(), Input{Paranoid: paranoid, Topics: []string{"Gym edits"}, Items: items})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 judgements (one failed), got %d", len(got))
	}
	if got[0].SpoilerOf != "Black Clover" || got[0].Topic != "Anime and manga" {
		t.Errorf("first item: %+v", got[0])
	}
	if got[1].SpoilerOf != "" {
		t.Errorf("below threshold must not count as a spoiler: %+v", got[1])
	}
}

func TestJevJudgeAllFailed(t *testing.T) {
	asker := fakeAsker{answer: func(string, map[string]jev.Question) (map[string]jev.Answer, error) {
		return nil, errors.New("down")
	}}
	_, err := NewJevJudge(asker).Judge(context.Background(), Input{Items: []domain.CheckItem{{ID: "a", Site: "x", Title: "t"}}})
	if err == nil {
		t.Fatal("want an error when every item failed")
	}
}

func TestTopicChoices(t *testing.T) {
	got := topicChoices([]string{"Gym edits", "gym edits", "Channel: Foo", "  Anime and manga "})
	if got[0] != "Gym edits" || got[1] != "Anime and manga" {
		t.Fatalf("existing topics first and de-duplicated: %v", got[:3])
	}
	if len(got) > maxTopicChoices {
		t.Fatalf("too many choices: %d", len(got))
	}
	for _, topic := range got {
		if strings.HasPrefix(strings.ToLower(topic), "channel:") {
			t.Fatalf("channel fallback topics must not be offered: %v", topic)
		}
	}
}
