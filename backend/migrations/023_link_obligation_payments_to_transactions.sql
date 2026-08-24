-- +goose Up

ALTER TABLE obligation_payments
    ADD COLUMN transaction_id BIGINT UNIQUE REFERENCES transactions(id) ON DELETE CASCADE;

INSERT INTO categories(workspace_id,name,type,color,icon,expense_class)
SELECT w.id,'Penerimaan piutang','income','#7f9d8e','hand-coins',NULL
FROM workspaces w
ON CONFLICT (workspace_id,name,type) DO NOTHING;

-- +goose Down

ALTER TABLE obligation_payments DROP COLUMN IF EXISTS transaction_id;
