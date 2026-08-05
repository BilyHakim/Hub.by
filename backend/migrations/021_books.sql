-- +goose Up

CREATE TABLE book_titles (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    catalog_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    author TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    cover_url TEXT NOT NULL DEFAULT '',
    publish_year SMALLINT CHECK (publish_year IS NULL OR publish_year BETWEEN 1000 AND 2200),
    total_pages INTEGER NOT NULL CHECK (total_pages BETWEEN 1 AND 100000),
    status TEXT NOT NULL DEFAULT 'planned' CHECK (status IN ('planned', 'reading', 'completed', 'dropped')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, workspace_id)
);

CREATE TABLE reading_sessions (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title_id BIGINT NOT NULL,
    read_at DATE NOT NULL DEFAULT CURRENT_DATE,
    start_page INTEGER NOT NULL CHECK (start_page >= 0),
    end_page INTEGER NOT NULL CHECK (end_page > start_page),
    pages_read INTEGER NOT NULL CHECK (pages_read > 0),
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (title_id, workspace_id) REFERENCES book_titles(id, workspace_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX book_titles_workspace_catalog_idx
    ON book_titles(workspace_id, catalog_id) WHERE catalog_id <> '';
CREATE INDEX book_titles_workspace_status_idx ON book_titles(workspace_id, status, updated_at DESC);
CREATE INDEX reading_sessions_workspace_date_idx ON reading_sessions(workspace_id, read_at DESC, id DESC);
CREATE INDEX reading_sessions_title_date_idx ON reading_sessions(title_id, read_at DESC, id DESC);

-- +goose Down

DROP TABLE IF EXISTS reading_sessions;
DROP TABLE IF EXISTS book_titles;
