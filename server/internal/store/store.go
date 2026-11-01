package store

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound   = errors.New("store: not found")
	ErrDuplicate  = errors.New("store: webhook already processed")
	ErrTrialSpent = errors.New("store: trial allowance used up")
)

type Counter int

const (
	CounterVoice Counter = iota
	CounterChecks
)

type User struct {
	ID         int64
	ClerkID    string
	Email      string
	VoiceUsed  int
	ChecksUsed int
	LicensedAt *time.Time
	CreatedAt  time.Time
	LastSeenAt time.Time
}

func (u User) Active() bool { return u.LicensedAt != nil }

func (u User) Used(c Counter) int {
	if c == CounterVoice {
		return u.VoiceUsed
	}
	return u.ChecksUsed
}

const (
	CheckoutPending = "pending"
	CheckoutPaid    = "paid"
)

type CheckoutSession struct {
	ID                string
	UserID            int64
	ProviderSessionID *string
	Status            string
	CreatedAt         time.Time
	PaidAt            *time.Time
}

type Payment struct {
	Provider          string
	WebhookID         string
	EventType         string
	ProviderPaymentID string
	CheckoutSessionID string
	ClerkID           string
	Amount            int64
	Currency          string
	At                time.Time
}

type Store interface {
	Ping(ctx context.Context) error
	Close()

	UserByClerkID(ctx context.Context, clerkID string) (User, error)
	CreateUser(ctx context.Context, u User) (User, error)
	SetUserEmail(ctx context.Context, userID int64, email string) error
	TouchUser(ctx context.Context, userID int64, at time.Time) error

	SpendTrial(ctx context.Context, userID int64, c Counter, n, limit int) (int, error)
	RefundTrial(ctx context.Context, userID int64, c Counter, n int) error

	CreateCheckoutSession(ctx context.Context, s CheckoutSession) error
	SetCheckoutProviderID(ctx context.Context, id, providerSessionID string) error
	CheckoutSession(ctx context.Context, id string) (CheckoutSession, error)

	RecordPayment(ctx context.Context, p Payment) (bool, error)
}
