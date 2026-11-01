package clerkauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/go-jose/go-jose/v3"
	josejwt "github.com/go-jose/go-jose/v3/jwt"
)

const testKeyID = "ins_test_key"

type tokenClaims struct {
	josejwt.Claims
	AuthorizedParty string `json:"azp,omitempty"`
}

func signer(t *testing.T) (*rsa.PrivateKey, jose.Signer) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	s, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", testKeyID))
	if err != nil {
		t.Fatal(err)
	}
	return key, s
}

func token(t *testing.T, s jose.Signer, subject, azp string, expires time.Time) string {
	t.Helper()
	now := time.Now()
	raw, err := josejwt.Signed(s).Claims(tokenClaims{
		Claims: josejwt.Claims{
			Issuer:    "https://example.clerk.accounts.dev",
			Subject:   subject,
			IssuedAt:  josejwt.NewNumericDate(now.Add(-time.Minute)),
			NotBefore: josejwt.NewNumericDate(now.Add(-time.Minute)),
			Expiry:    josejwt.NewNumericDate(expires),
		},
		AuthorizedParty: azp,
	}).CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestVerify(t *testing.T) {
	key, s := signer(t)
	_, other := signer(t)
	later := time.Now().Add(time.Minute)
	tests := []struct {
		name    string
		parties []string
		token   string
		want    string
	}{
		{"valid", nil, token(t, s, "user_a", "https://fuckalgo.app", later), "user_a"},
		{"allowed party", []string{"https://fuckalgo.app", "chrome-extension://abc"}, token(t, s, "user_a", "chrome-extension://abc", later), "user_a"},
		{"other party", []string{"https://fuckalgo.app"}, token(t, s, "user_a", "https://evil.example", later), ""},
		{"expired", nil, token(t, s, "user_a", "", time.Now().Add(-time.Hour)), ""},
		{"wrong key", nil, token(t, other, "user_a", "", later), ""},
		{"no subject", nil, token(t, s, "", "", later), ""},
		{"garbage", nil, "not a token", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New("sk_test_offline", tt.parties)
			c.keys.put(testKeyID, &clerk.JSONWebKey{Key: &key.PublicKey, KeyID: testKeyID, Algorithm: "RS256", Use: "sig"})
			got, err := c.Verify(context.Background(), tt.token)
			if got != tt.want || (tt.want == "") != (err != nil) {
				t.Fatalf("Verify() = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestAvailable(t *testing.T) {
	if New("", nil).Available() || !New("sk_test_offline", nil).Available() {
		t.Fatal("Available() must follow the secret key")
	}
}

func TestJWKCacheExpires(t *testing.T) {
	c := newJWKCache()
	now := time.Now()
	c.now = func() time.Time { return now }
	fetches := 0
	fetch := func(context.Context, string) (*clerk.JSONWebKey, error) {
		fetches++
		return &clerk.JSONWebKey{KeyID: testKeyID}, nil
	}
	for range 3 {
		if _, err := c.get(context.Background(), testKeyID, fetch); err != nil {
			t.Fatal(err)
		}
	}
	now = now.Add(jwkCacheTTL + time.Second)
	_, _ = c.get(context.Background(), testKeyID, fetch)
	if fetches != 2 {
		t.Fatalf("fetches = %d, want 2", fetches)
	}
	failing := func(context.Context, string) (*clerk.JSONWebKey, error) { return nil, errors.New("offline") }
	if _, err := c.get(context.Background(), "unknown", failing); err == nil {
		t.Fatal("a failed fetch must surface")
	}
}

func TestPrimaryEmail(t *testing.T) {
	primary := "idn_2"
	tests := []struct {
		name string
		user *clerk.User
		want string
		err  error
	}{
		{"primary wins", &clerk.User{PrimaryEmailAddressID: &primary, EmailAddresses: []*clerk.EmailAddress{
			{ID: "idn_1", EmailAddress: "old@example.com"}, {ID: "idn_2", EmailAddress: "main@example.com"}}}, "main@example.com", nil},
		{"first when no primary", &clerk.User{EmailAddresses: []*clerk.EmailAddress{{ID: "idn_1", EmailAddress: "a@example.com"}}}, "a@example.com", nil},
		{"none", &clerk.User{}, "", ErrNoEmail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := primaryEmail(tt.user)
			if got != tt.want || !errors.Is(err, tt.err) {
				t.Fatalf("primaryEmail() = %q, %v; want %q, %v", got, err, tt.want, tt.err)
			}
		})
	}
}
