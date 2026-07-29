-- +goose Up

ALTER TABLE categories
ADD COLUMN expense_class TEXT
CHECK (expense_class IN ('essential', 'obligation', 'discretionary', 'future'));

UPDATE categories
SET expense_class = CASE
    WHEN type = 'income' THEN NULL
    WHEN name IN ('Makanan', 'Transportasi', 'Tempat Tinggal') THEN 'essential'
    WHEN name IN ('Tagihan', 'Cicilan') THEN 'obligation'
    WHEN name IN ('Hiburan', 'Belanja') THEN 'discretionary'
    ELSE 'discretionary'
END;

CREATE INDEX categories_workspace_expense_class_idx
ON categories(workspace_id, expense_class);

-- +goose Down

DROP INDEX IF EXISTS categories_workspace_expense_class_idx;
ALTER TABLE categories DROP COLUMN expense_class;

