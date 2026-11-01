package svix

import (
	"encoding/base64"
	"errors"
	"strconv"
	"testing"
	"time"
)

func TestVerify(t *testing.T) {
	key := []byte("a-very-secret-signing-key")
	secret := "whsec_" + base64.StdEncoding.EncodeToString(key)
	now := time.Unix(1_800_000_000, 0)
	ts := strconv.FormatInt(now.Unix(), 10)
	payload := []byte(`{"type":"payment.succeeded"}`)
	good := "v1," + base64.StdEncoding.EncodeToString(Sign(key, "msg_1", ts, payload))
	stale := strconv.FormatInt(now.Add(-10*time.Minute).Unix(), 10)

	tests := []struct {
		name      string
		payload   []byte
		id        string
		timestamp string
		signature string
		secret    string
		want      error
	}{
		{"valid", payload, "msg_1", ts, good, secret, nil},
		{"valid among several", payload, "msg_1", ts, "v1,bm9wZQ== " + good, secret, nil},
		{"tampered payload", []byte(`{"type":"x"}`), "msg_1", ts, good, secret, ErrNoMatch},
		{"wrong id", payload, "msg_2", ts, good, secret, ErrNoMatch},
		{"wrong version", payload, "msg_1", ts, "v2" + good[2:], secret, ErrNoMatch},
		{"stale timestamp", payload, "msg_1", stale, good, secret, ErrInvalidTimestamp},
		{"garbage timestamp", payload, "msg_1", "abc", good, secret, ErrInvalidTimestamp},
		{"bad secret", payload, "msg_1", ts, good, "whsec_!!!", ErrInvalidSecret},
		{"empty secret", payload, "msg_1", ts, good, "", ErrInvalidSecret},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Verify(tt.payload, tt.id, tt.timestamp, tt.signature, tt.secret, now)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Verify() = %v, want %v", err, tt.want)
			}
		})
	}
}
