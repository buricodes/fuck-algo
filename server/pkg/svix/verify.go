package svix

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const tolerance = 5 * time.Minute

var (
	ErrInvalidSecret    = errors.New("svix: invalid signing secret")
	ErrInvalidTimestamp = errors.New("svix: invalid or expired timestamp")
	ErrNoMatch          = errors.New("svix: no matching signature")
)

func Verify(payload []byte, id, timestamp, signature, secret string, now time.Time) error {
	key, err := decodeSecret(secret)
	if err != nil {
		return err
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTimestamp, err)
	}
	if age := now.Sub(time.Unix(ts, 0)); age > tolerance || age < -tolerance {
		return fmt.Errorf("%w: outside %s tolerance", ErrInvalidTimestamp, tolerance)
	}
	expected := Sign(key, id, timestamp, payload)
	for _, part := range strings.Fields(signature) {
		version, sig, ok := strings.Cut(part, ",")
		if !ok || version != "v1" {
			continue
		}
		got, err := base64.StdEncoding.DecodeString(sig)
		if err == nil && hmac.Equal(got, expected) {
			return nil
		}
	}
	return ErrNoMatch
}

func Sign(key []byte, id, timestamp string, payload []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + timestamp + "."))
	mac.Write(payload)
	return mac.Sum(nil)
}

func CheckSecret(secret string) error {
	_, err := decodeSecret(secret)
	return err
}

func decodeSecret(secret string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSecret, err)
	}
	if len(decoded) == 0 {
		return nil, fmt.Errorf("%w: empty", ErrInvalidSecret)
	}
	return decoded, nil
}
