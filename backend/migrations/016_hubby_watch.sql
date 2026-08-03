-- +goose Up

CREATE TABLE watch_titles (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'series')),
    title TEXT NOT NULL,
    genre TEXT NOT NULL DEFAULT '',
    release_year SMALLINT CHECK (release_year IS NULL OR release_year BETWEEN 1888 AND 2200),
    runtime_minutes SMALLINT NOT NULL CHECK (runtime_minutes BETWEEN 1 AND 1440),
    total_episodes INTEGER CHECK (total_episodes IS NULL OR total_episodes > 0),
    status TEXT NOT NULL DEFAULT 'planned' CHECK (status IN ('planned', 'watching', 'completed', 'dropped')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, workspace_id)
);

CREATE TABLE watch_sessions (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title_id BIGINT NOT NULL,
    watched_at DATE NOT NULL DEFAULT CURRENT_DATE,
    duration_minutes SMALLINT NOT NULL CHECK (duration_minutes BETWEEN 1 AND 1440),
    season_number SMALLINT CHECK (season_number IS NULL OR season_number > 0),
    episode_number INTEGER CHECK (episode_number IS NULL OR episode_number > 0),
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (title_id, workspace_id) REFERENCES watch_titles(id, workspace_id) ON DELETE CASCADE
);

CREATE INDEX watch_titles_workspace_status_idx ON watch_titles(workspace_id, status, updated_at DESC);
CREATE INDEX watch_sessions_workspace_date_idx ON watch_sessions(workspace_id, watched_at DESC, id DESC);
CREATE INDEX watch_sessions_title_date_idx ON watch_sessions(title_id, watched_at DESC, id DESC);

-- +goose Down

DROP TABLE IF EXISTS watch_sessions;
DROP TABLE IF EXISTS watch_titles;
