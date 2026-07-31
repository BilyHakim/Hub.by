-- +goose Up

CREATE TABLE obligations (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('debt', 'receivable')),
    name TEXT NOT NULL,
    platform TEXT NOT NULL DEFAULT '',
    original_amount BIGINT NOT NULL CHECK (original_amount > 0),
    installment_count SMALLINT NOT NULL CHECK (installment_count BETWEEN 1 AND 360),
    start_date DATE NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE obligation_payments (
    id BIGSERIAL PRIMARY KEY,
    obligation_id BIGINT NOT NULL REFERENCES obligations(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL CHECK (amount > 0),
    paid_at DATE NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX obligations_workspace_type_idx ON obligations(workspace_id, type);
CREATE INDEX obligation_payments_obligation_idx ON obligation_payments(obligation_id, paid_at, id);

-- +goose Down

DROP TABLE IF EXISTS obligation_payments;
DROP TABLE IF EXISTS obligations;
