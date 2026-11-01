package checks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/buricodes/fuck-algo/server/internal/domain"
	"github.com/buricodes/fuck-algo/server/internal/prompts"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
)

type Model interface {
	Available() bool
	JSON(ctx context.Context, system, user string, schema json.RawMessage) ([]byte, error)
}

type ModelJudge struct {
	model Model
}

func NewModelJudge(model Model) *ModelJudge { return &ModelJudge{model: model} }

func (j *ModelJudge) Available() bool { return j.model.Available() }

func (j *ModelJudge) Judge(ctx context.Context, in Input) ([]Judgement, error) {
	user, err := json.Marshal(modelInput(in))
	if err != nil {
		return nil, fmt.Errorf("checks: encoding input: %w", err)
	}
	raw, err := j.model.JSON(ctx, prompts.Checks, string(user), schema)
	if err != nil {
		return nil, err
	}
	var out modelOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, &apperr.UpstreamError{Provider: "model", Err: fmt.Errorf("checks: decoding reply: %w", err)}
	}
	judgements := make([]Judgement, 0, len(out.Results))
	for _, r := range out.Results {
		judgements = append(judgements, Judgement{I: r.I, SpoilerOf: r.SpoilerOf, Topic: r.Topic})
	}
	return judgements, nil
}

var schema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "results": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "i": {"type": "integer"},
          "spoilerOf": {"type": "string"},
          "topic": {"type": "string"}
        },
        "required": ["i", "spoilerOf", "topic"],
        "additionalProperties": false
      }
    }
  },
  "required": ["results"],
  "additionalProperties": false
}`)

type watchInput struct {
	Name     string             `json:"name"`
	Type     domain.SpoilerType `json:"type"`
	Keywords []string           `json:"keywords"`
}

type itemInput struct {
	I       int         `json:"i"`
	Site    domain.Site `json:"site"`
	Title   string      `json:"title"`
	Channel string      `json:"channel,omitempty"`
	Text    string      `json:"text,omitempty"`
}

type input struct {
	Watchlist      []watchInput `json:"watchlist"`
	ExistingTopics []string     `json:"existingTopics"`
	Items          []itemInput  `json:"items"`
}

type modelOutput struct {
	Results []struct {
		I         int    `json:"i"`
		SpoilerOf string `json:"spoilerOf"`
		Topic     string `json:"topic"`
	} `json:"results"`
}

func modelInput(in Input) input {
	out := input{Watchlist: []watchInput{}, ExistingTopics: []string{}, Items: make([]itemInput, len(in.Items))}
	for _, w := range in.Paranoid {
		out.Watchlist = append(out.Watchlist, watchInput{Name: w.Name, Type: w.Type, Keywords: w.Keywords})
	}
	if in.Topics != nil {
		out.ExistingTopics = in.Topics
	}
	for i, item := range in.Items {
		out.Items[i] = itemInput{I: i, Site: item.Site, Title: item.Title, Channel: item.Channel, Text: item.Text}
	}
	return out
}
