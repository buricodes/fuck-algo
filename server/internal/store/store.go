package store

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound    = errors.New("store: not found")
	ErrLicenceFull = errors.New("store: licence has no free install slots")
	ErrDuplicate   = errors.New("store: webhook already processed")
	ErrTrialSpent  = errors.New("store: trial allowance used up")
)

type Counter int

const (
	CounterVoice Counter = iota
	CounterChecks
)

type Install struct {
	ID            string
	TokenHash     string
	Version       string
	Browser       string
	LicenceID     *string
	LicenceActive bool
	KeyHint       *string
	VoiceUsed     int
	ChecksUsed    int
	CreatedAt     time.Time
	LastSeenAt    time.Time
}

func (i Install) Used(c Counter) int {
	if c == CounterVoice {
		return i.VoiceUsed
	}
	return i.ChecksUsed
}

type Licence struct {
	ID        string
	KeyHash   string
	KeyHint   string
	Status    string
	CreatedAt time.Time
}

const LicenceStatusActive = "active"

const (
	CheckoutPending = "pending"
	CheckoutPaid    = "paid"
)

type CheckoutSession struct {
	ID                string
	InstallID         *string
	ProviderSessionID *string
	Status            string
	LicenceID         *string
	RevealKey         *string
	CreatedAt         time.Time
	PaidAt            *time.Time
}

type Payment struct {
	Provider          string
	WebhookID         string
	EventType         string
	ProviderPaymentID string
	CheckoutSessionID string
	InstallID         string
	Email             string
	Amount            int64
	Currency          string
	LicenceID         string
	KeyHash           string
	KeyHint           string
	RevealKey         string
	MaxInstalls       int
	At                time.Time
}

type Store interface {
	Ping(ctx context.Context) error
	Close()

	CreateInstall(ctx context.Context, in Install) error
	InstallByTokenHash(ctx context.Context, tokenHash string) (Install, error)
	InstallExists(ctx context.Context, id string) (bool, error)
	TouchInstall(ctx context.Context, id string, at time.Time) error

	SpendTrial(ctx context.Context, installID string, c Counter, n, limit int) (int, error)
	RefundTrial(ctx context.Context, installID string, c Counter, n int) error

	LicenceByKeyHash(ctx context.Context, keyHash string) (Licence, error)
	AttachLicence(ctx context.Context, installID, licenceID string, maxInstalls int) error

	CreateCheckoutSession(ctx context.Context, s CheckoutSession) error
	SetCheckoutProviderID(ctx context.Context, id, providerSessionID string) error
	CheckoutSession(ctx context.Context, id string) (CheckoutSession, error)
	ClearRevealKeys(ctx context.Context, createdBefore time.Time) error

	RecordPayment(ctx context.Context, p Payment) error
}
