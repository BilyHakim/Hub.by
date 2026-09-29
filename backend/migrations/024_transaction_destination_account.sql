-- +goose Up

ALTER TABLE transactions
ADD COLUMN destination_account_id BIGINT REFERENCES accounts(id),
ADD CONSTRAINT transactions_destination_requires_expense
  CHECK (destination_account_id IS NULL OR type = 'expense'),
ADD CONSTRAINT transactions_destination_differs_from_source
  CHECK (destination_account_id IS NULL OR destination_account_id <> account_id);

CREATE INDEX transactions_destination_account_idx
ON transactions(destination_account_id) WHERE destination_account_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS transactions_destination_account_idx;
ALTER TABLE transactions DROP CONSTRAINT transactions_destination_differs_from_source;
ALTER TABLE transactions DROP CONSTRAINT transactions_destination_requires_expense;
ALTER TABLE transactions DROP COLUMN destination_account_id;
