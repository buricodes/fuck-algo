package jev

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/buricodes/fuck-algo/server/internal/config"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/httpx"
	"github.com/buricodes/fuck-algo/server/pkg/slots"
)

const (
	endpoint      = "https://openrouter.ai/api/v1/systemone"
	provider      = "openrouter"
	maxReplyBytes = 1 << 20
)

type Question struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria,omitempty"`
}

type Answer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice,omitempty"`
	Noul          float64            `json:"noul,omitempty"`
	Score         float64            `json:"score,omitempty"`
	Confidence    float64            `json:"confidence,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
}

type request struct {
	Model     string              `json:"model"`
	State     string              `json:"state"`
	Questions map[string]Question `json:"questions"`
}

type usage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost"`
}

type response struct {
	Answers map[string]Answer `json:"answers"`
	Usage   usage             `json:"usage"`
}

type Client struct {
	apiKey  string
	model   string
	timeout time.Duration
	sem     chan struct{}
}

func New(apiKey string, cfg config.Jev) *Client {
	return &Client{
		apiKey:  apiKey,
		model:   cfg.Model,
		timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		sem:     make(chan struct{}, max(1, cfg.MaxConcurrent)),
	}
}

func (c *Client) Available() bool { return c.apiKey != "" }

func (c *Client) Ask(ctx context.Context, state string, questions map[string]Question) (map[string]Answer, error) {
	release, err := slots.Acquire(ctx, c.sem)
	if err != nil {
		return nil, err
	}
	defer release()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	payload, err := json.Marshal(request{Model: c.model, State: state, Questions: questions})
	if err != nil {
		return nil, fmt.Errorf("jev: encoding request: %w", err)
	}
	start := time.Now()
	resp, err := httpx.Do(ctx, httpx.Request{
		Provider: provider,
		Method:   http.MethodPost,
		URL:      endpoint,
		Header:   map[string]string{"Content-Type": "application/json", "Authorization": "Bearer " + c.apiKey},
		Body:     payload,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := httpx.ReadLimited(resp.Body, maxReplyBytes)
	if err != nil {
		return nil, &apperr.UpstreamError{Provider: provider, Err: err}
	}
	var parsed response
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, &apperr.UpstreamError{Provider: provider, Err: fmt.Errorf("jev: decoding reply: %w", err)}
	}
	slog.InfoContext(ctx, "jev usage", "model", c.model, "questions", len(questions),
		"inputTokens", parsed.Usage.InputTokens, "outputTokens", parsed.Usage.OutputTokens,
		"cost", parsed.Usage.Cost, "ms", time.Since(start).Milliseconds())
	return parsed.Answers, nil
}
