package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

var _ Store = (*Postgres)(nil)

func OpenPostgres(ctx context.Context, url string, maxConns int32) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parsing DATABASE_URL: %w", err)
	}
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	cfg.MaxConnIdleTime = 3 * time.Minute
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("reaching database: %w", err)
	}
	if err := Migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }
func (p *Postgres) Close()                         { p.pool.Close() }

const userColumns = `id, clerk_id, email, voice_used, checks_used, licensed_at, created_at, last_seen_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.ClerkID, &u.Email, &u.VoiceUsed, &u.ChecksUsed, &u.LicensedAt, &u.CreatedAt, &u.LastSeenAt)
	return u, notFound(err)
}

func (p *Postgres) UserByClerkID(ctx context.Context, clerkID string) (User, error) {
	return scanUser(p.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE clerk_id = $1`, clerkID))
}

func (p *Postgres) CreateUser(ctx context.Context, u User) (User, error) {
	return scanUser(p.pool.QueryRow(ctx, `
		INSERT INTO users (clerk_id, email, created_at, last_seen_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (clerk_id) DO UPDATE SET clerk_id = EXCLUDED.clerk_id
		RETURNING `+userColumns, u.ClerkID, u.Email, u.CreatedAt, u.LastSeenAt))
}

func (p *Postgres) SetUserEmail(ctx context.Context, userID int64, email string) error {
	tag, err := p.pool.Exec(ctx, `UPDATE users SET email = $2 WHERE id = $1`, userID, email)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (p *Postgres) TouchUser(ctx context.Context, userID int64, at time.Time) error {
	_, err := p.pool.Exec(ctx, `UPDATE users SET last_seen_at = $2 WHERE id = $1`, userID, at)
	return err
}

var (
	spendSQL = map[Counter]string{
		CounterVoice:  spendStatement("voice_used"),
		CounterChecks: spendStatement("checks_used"),
	}
	refundSQL = map[Counter]string{
		CounterVoice:  `UPDATE users SET voice_used = GREATEST(voice_used - $2, 0) WHERE id = $1`,
		CounterChecks: `UPDATE users SET checks_used = GREATEST(checks_used - $2, 0) WHERE id = $1`,
	}
)

func spendStatement(column string) string {
	return `
		WITH before AS (SELECT id, ` + column + ` AS used FROM users WHERE id = $1 FOR UPDATE)
		UPDATE users u SET ` + column + ` = LEAST(before.used + $2, $3)
		FROM before
		WHERE u.id = before.id AND before.used < $3
		RETURNING u.` + column + ` - before.used`
}

func (p *Postgres) SpendTrial(ctx context.Context, userID int64, c Counter, n, limit int) (int, error) {
	var granted int
	err := p.pool.QueryRow(ctx, spendSQL[c], userID, n, limit).Scan(&granted)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrTrialSpent
	}
	return granted, err
}

func (p *Postgres) RefundTrial(ctx context.Context, userID int64, c Counter, n int) error {
	_, err := p.pool.Exec(ctx, refundSQL[c], userID, n)
	return err
}

func (p *Postgres) CreateCheckoutSession(ctx context.Context, s CheckoutSession) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO checkout_sessions (id, user_id, status, created_at) VALUES ($1, $2, $3, $4)`,
		s.ID, s.UserID, s.Status, s.CreatedAt)
	return err
}

func (p *Postgres) SetCheckoutProviderID(ctx context.Context, id, providerSessionID string) error {
	_, err := p.pool.Exec(ctx, `UPDATE checkout_sessions SET provider_session_id = $2 WHERE id = $1`, id, providerSessionID)
	return err
}

func (p *Postgres) CheckoutSession(ctx context.Context, id string) (CheckoutSession, error) {
	var s CheckoutSession
	err := p.pool.QueryRow(ctx, `
		SELECT id, user_id, provider_session_id, status, created_at, paid_at
		FROM checkout_sessions WHERE id = $1`, id).Scan(
		&s.ID, &s.UserID, &s.ProviderSessionID, &s.Status, &s.CreatedAt, &s.PaidAt)
	return s, notFound(err)
}

func (p *Postgres) RecordPayment(ctx context.Context, pay Payment) (bool, error) {
	matched := false
	err := pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			INSERT INTO webhook_events (provider, webhook_id, event_type, received_at) VALUES ($1, $2, $3, $4)
			ON CONFLICT (webhook_id) DO NOTHING`, pay.Provider, pay.WebhookID, pay.EventType, pay.At)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrDuplicate
		}
		userID, err := payer(ctx, tx, pay)
		if err != nil {
			return err
		}
		matched = userID != nil
		tag, err = tx.Exec(ctx, `
			INSERT INTO payments (user_id, provider_payment_id, amount, currency, created_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (provider_payment_id) DO NOTHING`,
			userID, pay.ProviderPaymentID, pay.Amount, pay.Currency, pay.At)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		return activate(ctx, tx, pay, userID)
	})
	return matched, err
}

func payer(ctx context.Context, tx pgx.Tx, pay Payment) (*int64, error) {
	var userID int64
	err := tx.QueryRow(ctx, `SELECT user_id FROM checkout_sessions WHERE id = $1`, pay.CheckoutSessionID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) && pay.ClerkID != "" {
		err = tx.QueryRow(ctx, `SELECT id FROM users WHERE clerk_id = $1`, pay.ClerkID).Scan(&userID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &userID, nil
}

func activate(ctx context.Context, tx pgx.Tx, pay Payment, userID *int64) error {
	if _, err := tx.Exec(ctx, `
		UPDATE checkout_sessions SET status = 'paid', paid_at = $2 WHERE id = $1`,
		pay.CheckoutSessionID, pay.At); err != nil {
		return err
	}
	if userID == nil {
		return nil
	}
	_, err := tx.Exec(ctx, `UPDATE users SET licensed_at = COALESCE(licensed_at, $2) WHERE id = $1`, *userID, pay.At)
	return err
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
