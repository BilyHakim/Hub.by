-- +goose Up

ALTER TABLE workspace_finance_settings
ADD COLUMN period_mode TEXT NOT NULL DEFAULT 'fixed_day'
CHECK (period_mode IN ('fixed_day', 'end_of_month'));

-- +goose Down

ALTER TABLE workspace_finance_settings
DROP COLUMN IF EXISTS period_mode;
