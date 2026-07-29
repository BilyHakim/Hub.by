-- +goose Up

CREATE TABLE workspace_finance_settings (
    workspace_id BIGINT PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    period_start_day SMALLINT NOT NULL DEFAULT 1 CHECK (period_start_day BETWEEN 1 AND 31),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO workspace_finance_settings(workspace_id,period_start_day)
SELECT id,1 FROM workspaces
ON CONFLICT (workspace_id) DO NOTHING;

-- +goose Down

DROP TABLE IF EXISTS workspace_finance_settings;

