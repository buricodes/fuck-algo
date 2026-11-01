package licence

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const Alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

const (
	keyPrefix   = "FA"
	keyGroups   = 4
	groupLength = 4
	keyBodyLen  = keyGroups * groupLength
	hintLength  = 4

	TokenPrefix = "fa_inst_"
	tokenBytes  = 32
)

var ErrMalformedKey = errors.New("licence key is malformed")

func NewKey() (string, error) {
	raw := make([]byte, keyBodyLen)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("reading randomness: %w", err)
	}
	body := make([]byte, keyBodyLen)
	for i, b := range raw {
		body[i] = Alphabet[b&31]
	}
	return format(string(body)), nil
}

func NormaliseKey(input string) (string, error) {
	compact := strings.ToUpper(strings.Map(func(r rune) rune {
		if r == '-' || r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, input))
	if len(compact) == len(keyPrefix)+keyBodyLen && strings.HasPrefix(compact, keyPrefix) {
		compact = compact[len(keyPrefix):]
	}
	if len(compact) != keyBodyLen || strings.Trim(compact, Alphabet) != "" {
		return "", ErrMalformedKey
	}
	return format(compact), nil
}

func HashKey(canonical string) string { return hash(canonical) }

func Hint(canonical string) string { return canonical[len(canonical)-hintLength:] }

func NewInstallToken() (string, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("reading randomness: %w", err)
	}
	return TokenPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

func HashToken(token string) string { return hash(token) }

func NewCheckoutSessionID() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("reading randomness: %w", err)
	}
	return "cks_" + base64.RawURLEncoding.EncodeToString(raw), nil
}

func format(body string) string {
	groups := make([]string, 0, keyGroups+1)
	groups = append(groups, keyPrefix)
	for i := 0; i < keyBodyLen; i += groupLength {
		groups = append(groups, body[i:i+groupLength])
	}
	return strings.Join(groups, "-")
}

func hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
