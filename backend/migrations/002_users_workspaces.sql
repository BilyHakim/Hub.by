-- +goose Up

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    display_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    initials VARCHAR(4) NOT NULL,
    subtitle TEXT NOT NULL DEFAULT 'Rencana bersama',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workspaces (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    initials VARCHAR(4) NOT NULL,
    owner_user_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workspace_members (
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, user_id)
);

ALTER TABLE users ADD COLUMN current_workspace_id BIGINT REFERENCES workspaces(id);

INSERT INTO users(display_name, email, initials, subtitle)
VALUES ('Bily & Ami', 'bily@hubby.local', 'BA', 'Rencana bersama');

INSERT INTO workspaces(name, initials, owner_user_id)
VALUES ('Keluarga AmBil', 'AB', 1);

INSERT INTO workspace_members(workspace_id, user_id, role)
VALUES (1, 1, 'owner');

UPDATE users SET current_workspace_id = 1 WHERE id = 1;

-- +goose Down

ALTER TABLE users DROP COLUMN IF EXISTS current_workspace_id;
DROP TABLE IF EXISTS workspace_members;
DROP TABLE IF EXISTS workspaces;
DROP TABLE IF EXISTS users;

