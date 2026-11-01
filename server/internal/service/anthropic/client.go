package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/buricodes/fuck-algo/server/internal/config"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/httpx"
	"github.com/buricodes/fuck-algo/server/pkg/slots"
)

const (
	messagesURL   = "https://api.anthropic.com/v1/messages"
	apiVersion    = "2023-06-01"
	provider      = "anthropic"
	maxReplyBytes = 4 << 20
)

type Model struct {
	label     string
	apiKey    string
	model     string
	thinking  string
	maxTokens int
	timeout   time.Duration
	sem       chan struct{}
}

func New(label, apiKey string, cfg config.Model) *Model {
	return &Model{
		label:     label,
		apiKey:    apiKey,
		model:     cfg.Model,
		thinking:  cfg.Thinking,
		maxTokens: cfg.MaxTokens,
		timeout:   time.Duration(cfg.TimeoutSeconds) * time.Second,
		sem:       make(chan struct{}, max(1, cfg.MaxConcurrent)),
	}
}

func (m *Model) Available() bool { return m.apiKey != "" }

func (m *Model) JSON(ctx context.Context, system, user string, schema json.RawMessage) ([]byte, error) {
	release, err := slots.Acquire(ctx, m.sem)
	if err != nil {
		return nil, err
	}
	defer release()
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	start := time.Now()
	reply, err := m.send(ctx, m.request(system, user, schema))
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "claude usage", "call", m.label, "model", m.model, "stopReason", reply.StopReason,
		"inputTokens", reply.Usage.InputTokens, "cacheReadInputTokens", reply.Usage.CacheReadInputTokens,
		"outputTokens", reply.Usage.OutputTokens, "ms", time.Since(start).Milliseconds())
	if reply.StopReason != "end_turn" {
		return nil, &apperr.UpstreamError{Provider: provider, Status: http.StatusOK, Body: "reply stopped early: " + reply.StopReason}
	}
	return []byte(reply.text()), nil
}

func (m *Model) request(system, user string, schema json.RawMessage) request {
	req := request{
		Model:        m.model,
		MaxTokens:    m.maxTokens,
		System:       []contentBlock{{Type: "text", Text: system, CacheControl: &cacheControl{Type: "ephemeral"}}},
		Messages:     []message{{Role: "user", Content: []contentBlock{{Type: "text", Text: user}}}},
		OutputConfig: &outputConfig{Format: &outputFormat{Type: "json_schema", Schema: schema}},
	}
	if m.thinking != "" {
		req.Thinking = &thinking{Type: m.thinking}
	}
	return req
}

func (m *Model) send(ctx context.Context, req request) (response, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return response{}, fmt.Errorf("anthropic: encoding request: %w", err)
	}
	resp, err := httpx.Do(ctx, httpx.Request{
		Provider: provider,
		Method:   http.MethodPost,
		URL:      messagesURL,
		Header: map[string]string{
			"Content-Type":      "application/json",
			"x-api-key":         m.apiKey,
			"anthropic-version": apiVersion,
		},
		Body: payload,
	})
	if err != nil {
		return response{}, err
	}
	defer resp.Body.Close()
	raw, err := httpx.ReadLimited(resp.Body, maxReplyBytes)
	if err != nil {
		return response{}, &apperr.UpstreamError{Provider: provider, Err: err}
	}
	var parsed response
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return response{}, &apperr.UpstreamError{Provider: provider, Err: fmt.Errorf("decoding reply: %w", err)}
	}
	return parsed, nil
}

type cacheControl struct {
	Type string `json:"type"`
}

type contentBlock struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}

type message struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

type thinking struct {
	Type string `json:"type"`
}

type outputFormat struct {
	Type   string          `json:"type"`
	Schema json.RawMessage `json:"schema"`
}

type outputConfig struct {
	Format *outputFormat `json:"format,omitempty"`
}

type request struct {
	Model        string         `json:"model"`
	MaxTokens    int            `json:"max_tokens"`
	System       []contentBlock `json:"system"`
	Messages     []message      `json:"messages"`
	Thinking     *thinking      `json:"thinking,omitempty"`
	OutputConfig *outputConfig  `json:"output_config,omitempty"`
}

type usage struct {
	InputTokens          int `json:"input_tokens"`
	OutputTokens         int `json:"output_tokens"`
	CacheReadInputTokens int `json:"cache_read_input_tokens"`
}

type response struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      usage  `json:"usage"`
}

func (r response) text() string {
	var b strings.Builder
	for _, block := range r.Content {
		if block.Type == "text" {
			b.WriteString(block.Text)
		}
	}
	return b.String()
}
