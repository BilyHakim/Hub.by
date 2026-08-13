-- +goose Up

ALTER TABLE watch_titles
    ADD COLUMN synopsis TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE watch_titles
    DROP COLUMN IF EXISTS synopsis;
