-- +goose Up

ALTER TABLE financial_goals
ADD COLUMN monthly_contribution BIGINT NOT NULL DEFAULT 0
CHECK (monthly_contribution >= 0);

-- +goose Down

ALTER TABLE financial_goals
DROP COLUMN IF EXISTS monthly_contribution;
