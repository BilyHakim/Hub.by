-- +goose Up

ALTER TABLE watch_titles
    ADD COLUMN imdb_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN poster_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN total_seasons SMALLINT CHECK (total_seasons IS NULL OR total_seasons > 0);

CREATE UNIQUE INDEX watch_titles_workspace_imdb_idx
    ON watch_titles(workspace_id, imdb_id) WHERE imdb_id <> '';

CREATE UNIQUE INDEX watch_sessions_episode_completion_idx
    ON watch_sessions(title_id, season_number, episode_number)
    WHERE season_number IS NOT NULL AND episode_number IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS watch_sessions_episode_completion_idx;
DROP INDEX IF EXISTS watch_titles_workspace_imdb_idx;
ALTER TABLE watch_titles
    DROP COLUMN IF EXISTS total_seasons,
    DROP COLUMN IF EXISTS poster_url,
    DROP COLUMN IF EXISTS imdb_id;
