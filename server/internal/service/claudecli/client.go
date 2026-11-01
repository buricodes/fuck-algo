package claudecli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/buricodes/fuck-algo/server/internal/config"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/slots"
)

const provider = "claude-cli"

type Model struct {
	label   string
	binary  string
	model   string
	timeout time.Duration
	sem     chan struct{}
}

func New(label string, cfg config.ClaudeCLI) *Model {
	binary, err := exec.LookPath(cfg.Binary)
	if err != nil {
		binary = ""
	}
	return &Model{
		label:   label,
		binary:  binary,
		model:   cfg.Model,
		timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		sem:     make(chan struct{}, max(1, cfg.MaxConcurrent)),
	}
}

func (m *Model) Available() bool { return m.binary != "" }

type output struct {
	IsError          bool            `json:"is_error"`
	Subtype          string          `json:"subtype"`
	Result           string          `json:"result"`
	StructuredOutput json.RawMessage `json:"structured_output"`
	TotalCostUSD     float64         `json:"total_cost_usd"`
}

func (m *Model) JSON(ctx context.Context, system, user string, schema json.RawMessage) ([]byte, error) {
	release, err := slots.Acquire(ctx, m.sem)
	if err != nil {
		return nil, err
	}
	defer release()
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, m.binary,
		"-p",
		"--model", m.model,
		"--output-format", "json",
		"--json-schema", string(schema),
		"--system-prompt", system,
		"--tools", "",
		"--no-session-persistence",
		"--strict-mcp-config",
		"--setting-sources", "",
	)
	cmd.Stdin = strings.NewReader(user)
	cmd.Env = withoutAPIKey(os.Environ())
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	start := time.Now()
	if err := cmd.Run(); err != nil {
		return nil, &apperr.UpstreamError{Provider: provider, Err: fmt.Errorf("%w: %s", err, lastLine(stderr.String()))}
	}
	var out output
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, &apperr.UpstreamError{Provider: provider, Err: fmt.Errorf("decoding reply: %w", err)}
	}
	slog.InfoContext(ctx, "claude cli usage", "call", m.label, "model", m.model,
		"costUSD", out.TotalCostUSD, "ms", time.Since(start).Milliseconds())
	if out.IsError || out.Subtype != "success" {
		return nil, &apperr.UpstreamError{Provider: provider, Body: lastLine(out.Result)}
	}
	if len(out.StructuredOutput) > 0 && string(out.StructuredOutput) != "null" {
		return out.StructuredOutput, nil
	}
	if out.Result != "" {
		return []byte(out.Result), nil
	}
	return nil, &apperr.UpstreamError{Provider: provider, Err: errors.New("empty reply")}
}

func withoutAPIKey(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if !strings.HasPrefix(kv, "ANTHROPIC_API_KEY=") {
			out = append(out, kv)
		}
	}
	return out
}

func lastLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndexByte(s, '\n'); i >= 0 {
		s = s[i+1:]
	}
	if len(s) > 300 {
		s = s[:300]
	}
	return s
}
