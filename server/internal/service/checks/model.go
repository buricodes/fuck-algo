package checks

import (
	"encoding/json"

	"github.com/buricodes/fuck-algo/server/internal/domain"
)

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

func modelInput(p *Plan, items []domain.CheckItem) input {
	in := input{Watchlist: []watchInput{}, ExistingTopics: []string{}, Items: make([]itemInput, len(items))}
	for _, w := range p.paranoid {
		in.Watchlist = append(in.Watchlist, watchInput{Name: w.Name, Type: w.Type, Keywords: w.Keywords})
	}
	if p.topics != nil {
		in.ExistingTopics = p.topics
	}
	for i, item := range items {
		in.Items[i] = itemInput{I: i, Site: item.Site, Title: item.Title, Channel: item.Channel, Text: item.Text}
	}
	return in
}
