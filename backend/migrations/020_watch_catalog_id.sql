-- +goose Up

DROP INDEX IF EXISTS watch_titles_workspace_imdb_idx;
ALTER TABLE watch_titles RENAME COLUMN imdb_id TO catalog_id;
CREATE UNIQUE INDEX watch_titles_workspace_catalog_idx
    ON watch_titles(workspace_id, catalog_id) WHERE catalog_id <> '';

-- +goose Down

DROP INDEX IF EXISTS watch_titles_workspace_catalog_idx;
ALTER TABLE watch_titles RENAME COLUMN catalog_id TO imdb_id;
CREATE UNIQUE INDEX watch_titles_workspace_imdb_idx
    ON watch_titles(workspace_id, imdb_id) WHERE imdb_id <> '';
