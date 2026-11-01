CREATE TABLE licences (
    id                  uuid PRIMARY KEY,
    key_hash            text NOT NULL UNIQUE,
    key_hint            text NOT NULL,
    email               text,
    provider_payment_id text UNIQUE,
    amount              bigint NOT NULL DEFAULT 0,
    currency            text NOT NULL DEFAULT '',
    status              text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE installs (
    id           uuid PRIMARY KEY,
    token_hash   text NOT NULL UNIQUE,
    licence_id   uuid REFERENCES licences (id) ON DELETE SET NULL,
    version      text NOT NULL,
    browser      text NOT NULL,
    voice_used   integer NOT NULL DEFAULT 0 CHECK (voice_used >= 0),
    checks_used  integer NOT NULL DEFAULT 0 CHECK (checks_used >= 0),
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX installs_licence_id_idx ON installs (licence_id);

CREATE TABLE checkout_sessions (
    id                  text PRIMARY KEY,
    install_id          uuid REFERENCES installs (id) ON DELETE SET NULL,
    provider_session_id text,
    status              text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid')),
    licence_id          uuid REFERENCES licences (id) ON DELETE SET NULL,
    reveal_key          text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    paid_at             timestamptz
);

CREATE INDEX checkout_sessions_reveal_idx ON checkout_sessions (created_at) WHERE reveal_key IS NOT NULL;

CREATE TABLE webhook_events (
    provider    text NOT NULL,
    webhook_id  text NOT NULL UNIQUE,
    event_type  text NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now()
);
