-- +goose Up

ALTER TABLE users DROP CONSTRAINT users_current_workspace_id_fkey;
ALTER TABLE users ADD CONSTRAINT users_current_workspace_id_fkey
    FOREIGN KEY (current_workspace_id) REFERENCES workspaces(id) ON DELETE SET NULL;

ALTER TABLE categories DROP CONSTRAINT categories_workspace_id_fkey;
ALTER TABLE categories ADD CONSTRAINT categories_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE accounts DROP CONSTRAINT accounts_workspace_id_fkey;
ALTER TABLE accounts ADD CONSTRAINT accounts_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE transactions DROP CONSTRAINT transactions_workspace_id_fkey;
ALTER TABLE transactions ADD CONSTRAINT transactions_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE monthly_budgets DROP CONSTRAINT monthly_budgets_workspace_id_fkey;
ALTER TABLE monthly_budgets ADD CONSTRAINT monthly_budgets_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE emergency_fund_settings DROP CONSTRAINT emergency_fund_settings_workspace_id_fkey;
ALTER TABLE emergency_fund_settings ADD CONSTRAINT emergency_fund_settings_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE financial_goals DROP CONSTRAINT financial_goals_workspace_id_fkey;
ALTER TABLE financial_goals ADD CONSTRAINT financial_goals_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE investments DROP CONSTRAINT investments_workspace_id_fkey;
ALTER TABLE investments ADD CONSTRAINT investments_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE pyramid_items DROP CONSTRAINT pyramid_items_workspace_id_fkey;
ALTER TABLE pyramid_items ADD CONSTRAINT pyramid_items_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE retirement_settings DROP CONSTRAINT retirement_settings_workspace_id_fkey;
ALTER TABLE retirement_settings ADD CONSTRAINT retirement_settings_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE account_transfers DROP CONSTRAINT account_transfers_workspace_id_fkey;
ALTER TABLE account_transfers ADD CONSTRAINT account_transfers_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;

-- +goose Down

ALTER TABLE account_transfers DROP CONSTRAINT account_transfers_workspace_id_fkey;
ALTER TABLE account_transfers ADD CONSTRAINT account_transfers_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id);
ALTER TABLE retirement_settings DROP CONSTRAINT retirement_settings_workspace_id_fkey;
ALTER TABLE retirement_settings ADD CONSTRAINT retirement_settings_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id);
ALTER TABLE pyramid_items DROP CONSTRAINT pyramid_items_workspace_id_fkey;
ALTER TABLE pyramid_items ADD CONSTRAINT pyramid_items_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id);
ALTER TABLE investments DROP CONSTRAINT investments_workspace_id_fkey;
ALTER TABLE investments ADD CONSTRAINT investments_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id);
ALTER TABLE financial_goals DROP CONSTRAINT financial_goals_workspace_id_fkey;
ALTER TABLE financial_goals ADD CONSTRAINT financial_goals_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id);
ALTER TABLE emergency_fund_settings DROP CONSTRAINT emergency_fund_settings_workspace_id_fkey;
ALTER TABLE emergency_fund_settings ADD CONSTRAINT emergency_fund_settings_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id);
ALTER TABLE monthly_budgets DROP CONSTRAINT monthly_budgets_workspace_id_fkey;
ALTER TABLE monthly_budgets ADD CONSTRAINT monthly_budgets_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id);
ALTER TABLE transactions DROP CONSTRAINT transactions_workspace_id_fkey;
ALTER TABLE transactions ADD CONSTRAINT transactions_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id);
ALTER TABLE accounts DROP CONSTRAINT accounts_workspace_id_fkey;
ALTER TABLE accounts ADD CONSTRAINT accounts_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id);
ALTER TABLE categories DROP CONSTRAINT categories_workspace_id_fkey;
ALTER TABLE categories ADD CONSTRAINT categories_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id);
ALTER TABLE users DROP CONSTRAINT users_current_workspace_id_fkey;
ALTER TABLE users ADD CONSTRAINT users_current_workspace_id_fkey FOREIGN KEY (current_workspace_id) REFERENCES workspaces(id);
