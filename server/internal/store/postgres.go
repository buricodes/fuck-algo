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

func (p *Postgres) CreateInstall(ctx context.Context, in Install) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO installs (id, token_hash, version, browser, created_at, last_seen_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $5)`,
		in.ID, in.TokenHash, in.Version, in.Browser, in.CreatedAt)
	return err
}

func (p *Postgres) InstallByTokenHash(ctx context.Context, tokenHash string) (Install, error) {
	var in Install
	err := p.pool.QueryRow(ctx, `
		SELECT i.id::text, i.token_hash, i.version, i.browser, i.licence_id::text,
		       COALESCE(l.status = 'active', false), l.key_hint,
		       i.voice_used, i.checks_used, i.created_at, i.last_seen_at
		FROM installs i LEFT JOIN licences l ON l.id = i.licence_id
		WHERE i.token_hash = $1`, tokenHash).Scan(
		&in.ID, &in.TokenHash, &in.Version, &in.Browser, &in.LicenceID,
		&in.LicenceActive, &in.KeyHint,
		&in.VoiceUsed, &in.ChecksUsed, &in.CreatedAt, &in.LastSeenAt)
	return in, notFound(err)
}

func (p *Postgres) InstallExists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM installs WHERE id = $1::uuid)`, id).Scan(&exists)
	return exists, err
}

func (p *Postgres) TouchInstall(ctx context.Context, id string, at time.Time) error {
	_, err := p.pool.Exec(ctx, `UPDATE installs SET last_seen_at = $2 WHERE id = $1::uuid`, id, at)
	return err
}

var (
	spendSQL = map[Counter]string{
		CounterVoice:  spendStatement("voice_used"),
		CounterChecks: spendStatement("checks_used"),
	}
	refundSQL = map[Counter]string{
		CounterVoice:  `UPDATE installs SET voice_used = GREATEST(voice_used - $2, 0) WHERE id = $1::uuid`,
		CounterChecks: `UPDATE installs SET checks_used = GREATEST(checks_used - $2, 0) WHERE id = $1::uuid`,
	}
)

func spendStatement(column string) string {
	return `
		WITH before AS (SELECT id, ` + column + ` AS used FROM installs WHERE id = $1::uuid FOR UPDATE)
		UPDATE installs i SET ` + column + ` = LEAST(before.used + $2, $3)
		FROM before
		WHERE i.id = before.id AND before.used < $3
		RETURNING i.` + column + ` - before.used`
}

func (p *Postgres) SpendTrial(ctx context.Context, installID string, c Counter, n, limit int) (int, error) {
	var granted int
	err := p.pool.QueryRow(ctx, spendSQL[c], installID, n, limit).Scan(&granted)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrTrialSpent
	}
	return granted, err
}

func (p *Postgres) RefundTrial(ctx context.Context, installID string, c Counter, n int) error {
	_, err := p.pool.Exec(ctx, refundSQL[c], installID, n)
	return err
}

func (p *Postgres) LicenceByKeyHash(ctx context.Context, keyHash string) (Licence, error) {
	var l Licence
	err := p.pool.QueryRow(ctx, `
		SELECT id::text, key_hash, key_hint, status, created_at FROM licences WHERE key_hash = $1`,
		keyHash).Scan(&l.ID, &l.KeyHash, &l.KeyHint, &l.Status, &l.CreatedAt)
	return l, notFound(err)
}

func (p *Postgres) AttachLicence(ctx context.Context, installID, licenceID string, maxInstalls int) error {
	return pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		return attachInTx(ctx, tx, installID, licenceID, maxInstalls)
	})
}

func attachInTx(ctx context.Context, tx pgx.Tx, installID, licenceID string, maxInstalls int) error {
	if _, err := tx.Exec(ctx, `SELECT 1 FROM licences WHERE id = $1::uuid FOR UPDATE`, licenceID); err != nil {
		return err
	}
	var already bool
	var taken int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(bool_or(id = $2::uuid), false), count(*) FROM installs WHERE licence_id = $1::uuid`,
		licenceID, installID).Scan(&already, &taken)
	if err != nil || already {
		return err
	}
	if taken >= maxInstalls {
		return ErrLicenceFull
	}
	tag, err := tx.Exec(ctx, `UPDATE installs SET licence_id = $2::uuid WHERE id = $1::uuid`, installID, licenceID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (p *Postgres) CreateCheckoutSession(ctx context.Context, s CheckoutSession) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO checkout_sessions (id, install_id, status, created_at) VALUES ($1, $2::uuid, $3, $4)`,
		s.ID, s.InstallID, s.Status, s.CreatedAt)
	return err
}

func (p *Postgres) SetCheckoutProviderID(ctx context.Context, id, providerSessionID string) error {
	_, err := p.pool.Exec(ctx, `UPDATE checkout_sessions SET provider_session_id = $2 WHERE id = $1`, id, providerSessionID)
	return err
}

func (p *Postgres) CheckoutSession(ctx context.Context, id string) (CheckoutSession, error) {
	var s CheckoutSession
	err := p.pool.QueryRow(ctx, `
		SELECT id, install_id::text, provider_session_id, status, licence_id::text, reveal_key, created_at, paid_at
		FROM checkout_sessions WHERE id = $1`, id).Scan(
		&s.ID, &s.InstallID, &s.ProviderSessionID, &s.Status, &s.LicenceID, &s.RevealKey, &s.CreatedAt, &s.PaidAt)
	return s, notFound(err)
}

func (p *Postgres) ClearRevealKeys(ctx context.Context, createdBefore time.Time) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE checkout_sessions SET reveal_key = NULL WHERE reveal_key IS NOT NULL AND created_at < $1`, createdBefore)
	return err
}

func (p *Postgres) RecordPayment(ctx context.Context, pay Payment) error {
	return pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			INSERT INTO webhook_events (provider, webhook_id, event_type, received_at) VALUES ($1, $2, $3, $4)
			ON CONFLICT (webhook_id) DO NOTHING`, pay.Provider, pay.WebhookID, pay.EventType, pay.At)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrDuplicate
		}
		created, err := insertLicence(ctx, tx, pay)
		if err != nil || !created {
			return err
		}
		if err := markCheckoutPaid(ctx, tx, pay); err != nil {
			return err
		}
		return attachPaidInstall(ctx, tx, pay)
	})
}

func insertLicence(ctx context.Context, tx pgx.Tx, pay Payment) (bool, error) {
	tag, err := tx.Exec(ctx, `
		INSERT INTO licences (id, key_hash, key_hint, email, provider_payment_id, amount, currency, status, created_at)
		VALUES ($1::uuid, $2, $3, NULLIF($4, ''), $5, $6, $7, 'active', $8)
		ON CONFLICT (provider_payment_id) DO NOTHING`,
		pay.LicenceID, pay.KeyHash, pay.KeyHint, pay.Email, pay.ProviderPaymentID, pay.Amount, pay.Currency, pay.At)
	return err == nil && tag.RowsAffected() == 1, err
}

func markCheckoutPaid(ctx context.Context, tx pgx.Tx, pay Payment) error {
	if pay.CheckoutSessionID == "" {
		return nil
	}
	_, err := tx.Exec(ctx, `
		UPDATE checkout_sessions SET status = 'paid', licence_id = $2::uuid, reveal_key = $3, paid_at = $4
		WHERE id = $1`, pay.CheckoutSessionID, pay.LicenceID, pay.RevealKey, pay.At)
	return err
}

func attachPaidInstall(ctx context.Context, tx pgx.Tx, pay Payment) error {
	if pay.InstallID == "" {
		return nil
	}
	err := attachInTx(ctx, tx, pay.InstallID, pay.LicenceID, pay.MaxInstalls)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
