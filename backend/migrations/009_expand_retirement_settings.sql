-- +goose Up

ALTER TABLE retirement_settings
ADD COLUMN monthly_contribution BIGINT NOT NULL DEFAULT 4000000
CHECK (monthly_contribution >= 0);

ALTER TABLE retirement_settings
ADD COLUMN withdrawal_rate NUMERIC(5,2) NOT NULL DEFAULT 4
CHECK (withdrawal_rate > 0 AND withdrawal_rate <= 100);

ALTER TABLE retirement_settings
ADD CONSTRAINT retirement_settings_workspace_key UNIQUE(workspace_id);

INSERT INTO retirement_settings(
    workspace_id,current_age,retirement_age,monthly_expense,inflation_rate,
    expected_return,current_fund,monthly_contribution,withdrawal_rate
)
SELECT id,25,55,5000000,3,6,100000000,4000000,4
FROM workspaces
ON CONFLICT(workspace_id) DO NOTHING;

-- +goose Down

ALTER TABLE retirement_settings
DROP CONSTRAINT IF EXISTS retirement_settings_workspace_key;

ALTER TABLE retirement_settings
DROP COLUMN IF EXISTS withdrawal_rate;

ALTER TABLE retirement_settings
DROP COLUMN IF EXISTS monthly_contribution;
