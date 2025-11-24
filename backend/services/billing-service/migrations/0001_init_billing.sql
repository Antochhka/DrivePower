CREATE TABLE IF NOT EXISTS tariffs (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price_per_kwh DOUBLE PRECISION NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS billing_transactions (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL,
    user_id BIGINT,
    energy_kwh DOUBLE PRECISION NOT NULL,
    price_per_kwh DOUBLE PRECISION NOT NULL,
    amount DOUBLE PRECISION NOT NULL,
    status TEXT NOT NULL DEFAULT 'completed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON billing_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_session_id ON billing_transactions(session_id);
