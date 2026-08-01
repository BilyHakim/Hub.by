-- +goose Up

CREATE TABLE account_transfers (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id),
    source_account_id BIGINT NOT NULL REFERENCES accounts(id),
    destination_account_id BIGINT NOT NULL REFERENCES accounts(id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    description TEXT NOT NULL DEFAULT '',
    occurred_at DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (source_account_id <> destination_account_id)
);

CREATE INDEX account_transfers_workspace_date_idx
    ON account_transfers(workspace_id, occurred_at, id);
CREATE INDEX account_transfers_source_account_idx
    ON account_transfers(source_account_id);
CREATE INDEX account_transfers_destination_account_idx
    ON account_transfers(destination_account_id);

-- +goose Down

DROP TABLE IF EXISTS account_transfers;
