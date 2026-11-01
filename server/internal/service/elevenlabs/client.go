package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/buricodes/fuck-algo/server/internal/config"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/httpx"
	"github.com/buricodes/fuck-algo/server/pkg/slots"
)

const (
	baseURL       = "https://api.elevenlabs.io"
	provider      = "elevenlabs"
	maxReplyBytes = 1 << 20
)

type Client struct {
	apiKey  string
	model   string
	timeout time.Duration
	sem     chan struct{}
}

func New(apiKey string, cfg config.ElevenLabs) *Client {
	return &Client{
		apiKey:  apiKey,
		model:   cfg.STTModel,
		timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		sem:     make(chan struct{}, max(1, cfg.MaxConcurrent)),
	}
}

func (c *Client) Available() bool { return c.apiKey != "" }

func (c *Client) RealtimeToken(ctx context.Context) (string, error) {
	var reply struct {
		Token string `json:"token"`
	}
	if err := c.post(ctx, "/v1/single-use-token/realtime_scribe", "", nil, &reply); err != nil {
		return "", err
	}
	if reply.Token == "" {
		return "", &apperr.UpstreamError{Provider: provider, Status: http.StatusOK, Body: "token response had no token"}
	}
	return reply.Token, nil
}

func (c *Client) Transcribe(ctx context.Context, audio []byte, filename string) (string, error) {
	body, contentType, err := transcriptionForm(c.model, audio, filename)
	if err != nil {
		return "", err
	}
	var reply struct {
		Text string `json:"text"`
	}
	if err := c.post(ctx, "/v1/speech-to-text", contentType, body, &reply); err != nil {
		return "", err
	}
	return reply.Text, nil
}

func (c *Client) post(ctx context.Context, path, contentType string, body []byte, into any) error {
	release, err := slots.Acquire(ctx, c.sem)
	if err != nil {
		return err
	}
	defer release()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	header := map[string]string{"xi-api-key": c.apiKey}
	if contentType != "" {
		header["Content-Type"] = contentType
	}
	resp, err := httpx.Do(ctx, httpx.Request{Provider: provider, Method: http.MethodPost, URL: baseURL + path, Header: header, Body: body})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := httpx.ReadLimited(resp.Body, maxReplyBytes)
	if err == nil {
		err = json.Unmarshal(raw, into)
	}
	if err != nil {
		return &apperr.UpstreamError{Provider: provider, Err: fmt.Errorf("reading %s reply: %w", path, err)}
	}
	return nil
}

func transcriptionForm(model string, audio []byte, filename string) ([]byte, string, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("model_id", model); err != nil {
		return nil, "", fmt.Errorf("elevenlabs: writing model_id: %w", err)
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", fmt.Errorf("elevenlabs: creating file field: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return nil, "", fmt.Errorf("elevenlabs: writing audio: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("elevenlabs: closing form: %w", err)
	}
	return body.Bytes(), writer.FormDataContentType(), nil
}
