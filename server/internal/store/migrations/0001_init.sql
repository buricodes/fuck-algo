CREATE TABLE users (
    id           bigserial PRIMARY KEY,
    clerk_id     text NOT NULL UNIQUE,
    email        text NOT NULL DEFAULT '',
    voice_used   integer NOT NULL DEFAULT 0 CHECK (voice_used >= 0),
    checks_used  integer NOT NULL DEFAULT 0 CHECK (checks_used >= 0),
    licensed_at  timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE checkout_sessions (
    id                  text PRIMARY KEY,
    user_id             bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider_session_id text,
    status              text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid')),
    created_at          timestamptz NOT NULL DEFAULT now(),
    paid_at             timestamptz
);

CREATE INDEX checkout_sessions_user_id_idx ON checkout_sessions (user_id);

CREATE TABLE payments (
    id                  bigserial PRIMARY KEY,
    user_id             bigint REFERENCES users (id) ON DELETE SET NULL,
    provider_payment_id text NOT NULL UNIQUE,
    amount              bigint NOT NULL DEFAULT 0,
    currency            text NOT NULL DEFAULT '',
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX payments_user_id_idx ON payments (user_id);

CREATE TABLE webhook_events (
    provider    text NOT NULL,
    webhook_id  text NOT NULL UNIQUE,
    event_type  text NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now()
);
