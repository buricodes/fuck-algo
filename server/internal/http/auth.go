package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/store"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/response"
)

const userKey = "user"

type Verifier interface {
	Available() bool
	Verify(ctx context.Context, token string) (string, error)
}

type Directory interface {
	PrimaryEmail(ctx context.Context, clerkID string) (string, error)
}

func (h *handler) requireUser(c *gin.Context) {
	if !h.Auth.Available() {
		response.Fail(c, unavailable("Sign in"))
		return
	}
	token, ok := bearerToken(c.GetHeader("Authorization"))
	if !ok {
		response.Fail(c, unauthorised())
		return
	}
	ctx := c.Request.Context()
	clerkID, err := h.Auth.Verify(ctx, token)
	if err != nil {
		slog.WarnContext(ctx, "session token rejected", "route", c.FullPath(), "error", err)
		response.Fail(c, unauthorised())
		return
	}
	u, err := h.userFor(ctx, clerkID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Set(userKey, u)
	c.Next()
}

func (h *handler) userFor(ctx context.Context, clerkID string) (store.User, error) {
	u, err := h.Store.UserByClerkID(ctx, clerkID)
	if !errors.Is(err, store.ErrNotFound) {
		return u, err
	}
	now := h.Now()
	u, err = h.Store.CreateUser(ctx, store.User{ClerkID: clerkID, Email: h.lookupEmail(ctx, clerkID), CreatedAt: now, LastSeenAt: now})
	if err == nil {
		slog.InfoContext(ctx, "user created", "user", u.ID, "hasEmail", u.Email != "")
	}
	return u, err
}

func (h *handler) lookupEmail(ctx context.Context, clerkID string) string {
	email, err := h.Users.PrimaryEmail(ctx, clerkID)
	if err != nil {
		slog.WarnContext(ctx, "reading the email from clerk failed, will retry later", "error", err)
		return ""
	}
	return email
}

func (h *handler) ensureEmail(ctx context.Context, u *store.User) error {
	if u.Email != "" {
		return nil
	}
	email := h.lookupEmail(ctx, u.ClerkID)
	if email == "" {
		return nil
	}
	if err := h.Store.SetUserEmail(ctx, u.ID, email); err != nil {
		return err
	}
	u.Email = email
	return nil
}

func bearerToken(header string) (string, bool) {
	token, ok := strings.CutPrefix(header, "Bearer ")
	token = strings.TrimSpace(token)
	return token, ok && token != ""
}

func currentUser(c *gin.Context) store.User {
	return c.MustGet(userKey).(store.User)
}

func byUser(c *gin.Context) string { return currentUser(c).ClerkID }

func unauthorised() error {
	return apperr.New(apperr.Unauthorised, "Sign in to use this.")
}
