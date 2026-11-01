package clerkauth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/clerk/clerk-sdk-go/v2/user"
)

const (
	jwkCacheTTL        = time.Hour
	jwkCacheMaxEntries = 16
	requestTimeout     = 10 * time.Second
	tokenLeeway        = 5 * time.Second
)

var (
	ErrNoSubject = errors.New("clerk: session token has no subject")
	ErrNoEmail   = errors.New("clerk: user has no email address")
)

type Clerk struct {
	jwks    *jwks.Client
	users   *user.Client
	parties map[string]bool
	keys    *jwkCache
}

func New(secretKey string, authorizedParties []string) *Clerk {
	c := &Clerk{parties: map[string]bool{}, keys: newJWKCache()}
	for _, party := range authorizedParties {
		if party = strings.TrimSpace(party); party != "" {
			c.parties[party] = true
		}
	}
	if secretKey == "" {
		return c
	}
	config := &clerk.ClientConfig{BackendConfig: clerk.BackendConfig{
		Key:        clerk.String(secretKey),
		HTTPClient: &http.Client{Timeout: requestTimeout},
	}}
	c.jwks = jwks.NewClient(config)
	c.users = user.NewClient(config)
	return c
}

func (c *Clerk) Available() bool { return c.jwks != nil }

func (c *Clerk) Verify(ctx context.Context, token string) (string, error) {
	unverified, err := jwt.Decode(ctx, &jwt.DecodeParams{Token: token})
	if err != nil {
		return "", err
	}
	jwk, err := c.keys.get(ctx, unverified.KeyID, c.fetchKey)
	if err != nil {
		return "", err
	}
	params := &jwt.VerifyParams{Token: token, JWK: jwk, Leeway: tokenLeeway}
	if len(c.parties) > 0 {
		params.AuthorizedPartyHandler = c.acceptsParty
	}
	claims, err := jwt.Verify(ctx, params)
	if err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", ErrNoSubject
	}
	return claims.Subject, nil
}

func (c *Clerk) acceptsParty(azp string) bool { return azp == "" || c.parties[azp] }

func (c *Clerk) fetchKey(ctx context.Context, keyID string) (*clerk.JSONWebKey, error) {
	return jwt.GetJSONWebKey(ctx, &jwt.GetJSONWebKeyParams{KeyID: keyID, JWKSClient: c.jwks})
}

func (c *Clerk) PrimaryEmail(ctx context.Context, clerkID string) (string, error) {
	u, err := c.users.Get(ctx, clerkID)
	if err != nil {
		return "", err
	}
	return primaryEmail(u)
}

func primaryEmail(u *clerk.User) (string, error) {
	var fallback string
	for _, address := range u.EmailAddresses {
		if address == nil || address.EmailAddress == "" {
			continue
		}
		if u.PrimaryEmailAddressID != nil && address.ID == *u.PrimaryEmailAddressID {
			return address.EmailAddress, nil
		}
		if fallback == "" {
			fallback = address.EmailAddress
		}
	}
	if fallback == "" {
		return "", ErrNoEmail
	}
	return fallback, nil
}

type cachedKey struct {
	key     *clerk.JSONWebKey
	expires time.Time
}

type jwkCache struct {
	mu   sync.Mutex
	keys map[string]cachedKey
	now  func() time.Time
}

func newJWKCache() *jwkCache {
	return &jwkCache{keys: map[string]cachedKey{}, now: time.Now}
}

func (c *jwkCache) get(ctx context.Context, keyID string, fetch func(context.Context, string) (*clerk.JSONWebKey, error)) (*clerk.JSONWebKey, error) {
	if key, ok := c.lookup(keyID); ok {
		return key, nil
	}
	key, err := fetch(ctx, keyID)
	if err != nil {
		return nil, err
	}
	c.put(keyID, key)
	return key, nil
}

func (c *jwkCache) lookup(keyID string) (*clerk.JSONWebKey, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.keys[keyID]
	if !ok || c.now().After(entry.expires) {
		return nil, false
	}
	return entry.key, true
}

func (c *jwkCache) put(keyID string, key *clerk.JSONWebKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.keys) >= jwkCacheMaxEntries {
		c.keys = map[string]cachedKey{}
	}
	c.keys[keyID] = cachedKey{key: key, expires: c.now().Add(jwkCacheTTL)}
}
