-- +goose Up

CREATE TABLE telegram_users (
    telegram_user_id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_id BIGINT NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    active_workspace_id BIGINT REFERENCES workspaces(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE telegram_pending_transactions (
    id BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT NOT NULL REFERENCES telegram_users(telegram_user_id) ON DELETE CASCADE,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    type transaction_type NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    description TEXT NOT NULL DEFAULT '',
    occurred_at DATE NOT NULL,
    category_id BIGINT REFERENCES categories(id) ON DELETE CASCADE,
    account_id BIGINT REFERENCES accounts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '30 minutes'
);

CREATE INDEX telegram_pending_user_idx
ON telegram_pending_transactions(telegram_user_id, created_at DESC);

-- +goose Down

DROP TABLE IF EXISTS telegram_pending_transactions;
DROP TABLE IF EXISTS telegram_users;
