-- +goose Up

CREATE TABLE obligation_payment_transactions (
    payment_id BIGINT NOT NULL REFERENCES obligation_payments(id) ON DELETE CASCADE,
    -- Workspace deletion cascades through both payments and transactions.
    -- Check their relationship after both cascades have completed.
    transaction_id BIGINT NOT NULL UNIQUE REFERENCES transactions(id) DEFERRABLE INITIALLY DEFERRED,
    PRIMARY KEY (payment_id, transaction_id)
);

INSERT INTO obligation_payment_transactions(payment_id,transaction_id)
SELECT id,transaction_id FROM obligation_payments WHERE transaction_id IS NOT NULL;

ALTER TABLE obligation_payments DROP COLUMN transaction_id;

-- +goose Down

-- Refuse a lossy rollback while payments have multiple funding sources.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM obligation_payment_transactions GROUP BY payment_id HAVING COUNT(*) > 1) THEN
        RAISE EXCEPTION 'Undo split payments before rolling back migration 026';
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE obligation_payments
    ADD COLUMN transaction_id BIGINT UNIQUE REFERENCES transactions(id) ON DELETE CASCADE;
UPDATE obligation_payments p SET transaction_id=l.transaction_id
FROM obligation_payment_transactions l WHERE l.payment_id=p.id;
DROP TABLE obligation_payment_transactions;
