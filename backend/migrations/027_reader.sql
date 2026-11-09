-- +goose Up
CREATE TABLE reader_books (
    title_id BIGINT PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    format TEXT NOT NULL CHECK (format IN ('pdf', 'epub')),
    content BYTEA NOT NULL CHECK (octet_length(content) BETWEEN 1 AND 52428800),
    location TEXT NOT NULL DEFAULT '',
    current_page INTEGER NOT NULL DEFAULT 0 CHECK (current_page >= 0),
    timer_enabled BOOLEAN NOT NULL DEFAULT false,
    last_read_at TIMESTAMPTZ,
    FOREIGN KEY (title_id, workspace_id) REFERENCES book_titles(id, workspace_id) ON DELETE CASCADE
);
CREATE TABLE reader_time (
    title_id BIGINT NOT NULL REFERENCES reader_books(title_id) ON DELETE CASCADE,
    session_id TEXT NOT NULL,
    seconds INTEGER NOT NULL CHECK (seconds BETWEEN 0 AND 86400),
    PRIMARY KEY (title_id, session_id)
);
CREATE TABLE reader_annotations (
    id BIGSERIAL PRIMARY KEY,
    title_id BIGINT NOT NULL REFERENCES reader_books(title_id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('highlight', 'bookmark', 'note')),
    location TEXT NOT NULL,
    page INTEGER NOT NULL CHECK (page > 0),
    quote TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    rects JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX reader_annotations_title_idx ON reader_annotations(title_id, created_at DESC);

-- +goose Down
DROP TABLE reader_annotations;
DROP TABLE reader_time;
DROP TABLE reader_books;
