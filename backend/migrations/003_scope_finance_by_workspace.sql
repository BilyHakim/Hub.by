-- +goose Up

ALTER TABLE categories ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE accounts ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE transactions ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE monthly_budgets ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE emergency_fund_settings ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE financial_goals ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE investments ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE pyramid_items ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE retirement_settings ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);

UPDATE categories SET workspace_id = 1;
UPDATE accounts SET workspace_id = 1;
UPDATE transactions SET workspace_id = 1;
UPDATE monthly_budgets SET workspace_id = 1;
UPDATE emergency_fund_settings SET workspace_id = 1;
UPDATE financial_goals SET workspace_id = 1;
UPDATE investments SET workspace_id = 1;
UPDATE pyramid_items SET workspace_id = 1;
UPDATE retirement_settings SET workspace_id = 1;

ALTER TABLE categories ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE accounts ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE transactions ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE monthly_budgets ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE emergency_fund_settings ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE financial_goals ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE investments ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE pyramid_items ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE retirement_settings ALTER COLUMN workspace_id SET NOT NULL;

ALTER TABLE categories DROP CONSTRAINT categories_name_type_key;
ALTER TABLE categories ADD CONSTRAINT categories_workspace_name_type_key UNIQUE(workspace_id, name, type);
ALTER TABLE accounts DROP CONSTRAINT accounts_name_key;
ALTER TABLE accounts ADD CONSTRAINT accounts_workspace_name_key UNIQUE(workspace_id, name);
ALTER TABLE monthly_budgets DROP CONSTRAINT monthly_budgets_category_id_month_key;
ALTER TABLE monthly_budgets ADD CONSTRAINT monthly_budgets_workspace_category_month_key UNIQUE(workspace_id, category_id, month);
ALTER TABLE emergency_fund_settings ADD CONSTRAINT emergency_fund_settings_workspace_key UNIQUE(workspace_id);

INSERT INTO categories(workspace_id,name,type,color,icon)
SELECT w.id, defaults.name, defaults.type::transaction_type, defaults.color, defaults.icon
FROM workspaces w
CROSS JOIN (VALUES
    ('Gaji','income','#49685c','briefcase'),
    ('Freelance','income','#7f9d8e','sparkles'),
    ('Makanan','expense','#e8a65d','utensils'),
    ('Transportasi','expense','#7894a0','car'),
    ('Tempat Tinggal','expense','#d77268','home'),
    ('Tagihan','expense','#9a8bb7','receipt'),
    ('Belanja','expense','#b4a464','shopping-bag'),
    ('Hiburan','expense','#638475','party-popper'),
    ('Cicilan','expense','#af685f','landmark')
) AS defaults(name,type,color,icon)
WHERE w.id <> 1
  AND NOT EXISTS (SELECT 1 FROM categories c WHERE c.workspace_id=w.id);

INSERT INTO accounts(workspace_id,name,kind,current_balance,is_emergency_fund)
SELECT w.id,'Rekening Utama','bank',0,FALSE
FROM workspaces w
WHERE w.id <> 1
  AND NOT EXISTS (SELECT 1 FROM accounts a WHERE a.workspace_id=w.id);

INSERT INTO emergency_fund_settings(workspace_id,monthly_expense,target_months)
SELECT w.id,0,6
FROM workspaces w
WHERE w.id <> 1
  AND NOT EXISTS (SELECT 1 FROM emergency_fund_settings e WHERE e.workspace_id=w.id);

CREATE INDEX transactions_workspace_date_idx ON transactions(workspace_id, occurred_at);
CREATE INDEX accounts_workspace_idx ON accounts(workspace_id);
CREATE INDEX goals_workspace_idx ON financial_goals(workspace_id);
CREATE INDEX investments_workspace_idx ON investments(workspace_id);

-- +goose Down

DROP INDEX IF EXISTS investments_workspace_idx;
DROP INDEX IF EXISTS goals_workspace_idx;
DROP INDEX IF EXISTS accounts_workspace_idx;
DROP INDEX IF EXISTS transactions_workspace_date_idx;

ALTER TABLE emergency_fund_settings DROP CONSTRAINT emergency_fund_settings_workspace_key;
ALTER TABLE monthly_budgets DROP CONSTRAINT monthly_budgets_workspace_category_month_key;
ALTER TABLE monthly_budgets ADD CONSTRAINT monthly_budgets_category_id_month_key UNIQUE(category_id, month);
ALTER TABLE accounts DROP CONSTRAINT accounts_workspace_name_key;
ALTER TABLE accounts ADD CONSTRAINT accounts_name_key UNIQUE(name);
ALTER TABLE categories DROP CONSTRAINT categories_workspace_name_type_key;
ALTER TABLE categories ADD CONSTRAINT categories_name_type_key UNIQUE(name, type);

ALTER TABLE retirement_settings DROP COLUMN workspace_id;
ALTER TABLE pyramid_items DROP COLUMN workspace_id;
ALTER TABLE investments DROP COLUMN workspace_id;
ALTER TABLE financial_goals DROP COLUMN workspace_id;
ALTER TABLE emergency_fund_settings DROP COLUMN workspace_id;
ALTER TABLE monthly_budgets DROP COLUMN workspace_id;
ALTER TABLE transactions DROP COLUMN workspace_id;
ALTER TABLE accounts DROP COLUMN workspace_id;
ALTER TABLE categories DROP COLUMN workspace_id;
