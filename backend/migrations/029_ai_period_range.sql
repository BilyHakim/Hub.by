-- +goose Up
ALTER TABLE ai_conversations ADD COLUMN start_month TEXT;
UPDATE ai_conversations
SET start_month = GREATEST('1900-01', to_char(to_date(month, 'YYYY-MM') - interval '3 months', 'YYYY-MM'));
ALTER TABLE ai_conversations ALTER COLUMN start_month SET NOT NULL;

-- +goose Down
ALTER TABLE ai_conversations DROP COLUMN start_month;
