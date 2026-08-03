-- +goose Up

ALTER TABLE watch_sessions ADD COLUMN is_backfill BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down

ALTER TABLE watch_sessions DROP COLUMN IF EXISTS is_backfill;
