package httpx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/buricodes/fuck-algo/server/pkg/apperr"
)

const (
	maxErrorBodyBytes  = 64 << 10
	retryDelay         = 500 * time.Millisecond
	minTimeLeftToRetry = 2 * time.Second
)

var ErrTooLarge = errors.New("response body too large")

var client = &http.Client{Transport: &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   20,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: time.Second,
}}

type Request struct {
	Provider string
	Method   string
	URL      string
	Header   map[string]string
	Body     []byte
}

func Do(ctx context.Context, r Request) (*http.Response, error) {
	resp, err := send(ctx, r)
	if err != nil {
		return nil, err
	}
	if isRetryable(resp.StatusCode) && hasTimeToRetry(ctx) {
		slog.WarnContext(ctx, "upstream busy, retrying once", "provider", r.Provider, "status", resp.StatusCode)
		discard(resp)
		select {
		case <-time.After(retryDelay):
		case <-ctx.Done():
			return nil, &apperr.UpstreamError{Provider: r.Provider, Err: ctx.Err()}
		}
		if resp, err = send(ctx, r); err != nil {
			return nil, err
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		return nil, &apperr.UpstreamError{Provider: r.Provider, Status: resp.StatusCode, Body: string(body)}
	}
	return resp, nil
}

func ReadLimited(body io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrTooLarge
	}
	return data, nil
}

func send(ctx context.Context, r Request) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, r.Method, r.URL, bytes.NewReader(r.Body))
	if err != nil {
		return nil, fmt.Errorf("%s: building request: %w", r.Provider, err)
	}
	for key, value := range r.Header {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &apperr.UpstreamError{Provider: r.Provider, Err: err}
	}
	return resp, nil
}

func isRetryable(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable || status == 529
}

func hasTimeToRetry(ctx context.Context) bool {
	deadline, ok := ctx.Deadline()
	return !ok || time.Until(deadline) > minTimeLeftToRetry
}

func discard(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBodyBytes))
	resp.Body.Close()
}
