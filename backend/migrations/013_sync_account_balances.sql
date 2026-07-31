-- +goose Up

-- Before automatic balance tracking existed, newly created accounts commonly
-- stayed at the default zero even when they already had transactions. Catch up
-- only those zero-balance accounts; preserve every balance adjusted manually.
WITH transaction_totals AS (
    SELECT
        workspace_id,
        account_id,
        COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE -amount END), 0)::bigint AS net_amount
    FROM transactions
    GROUP BY workspace_id, account_id
)
UPDATE accounts a
SET current_balance=t.net_amount,
    updated_at=now()
FROM transaction_totals t
WHERE a.workspace_id=t.workspace_id
  AND a.id=t.account_id
  AND a.current_balance=0
  AND t.net_amount<>0;

-- +goose Down

-- Data reconciliation is intentionally not reversed because doing so could
-- overwrite legitimate balance changes recorded after this migration.
