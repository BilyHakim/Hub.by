-- +goose Up
CREATE TABLE ai_credentials (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    provider TEXT NOT NULL CHECK (provider IN ('commandcode','openai','anthropic','ollama','custom')),
    base_url TEXT NOT NULL,
    api_format TEXT NOT NULL CHECK (api_format IN ('chat_completions','messages','ollama')),
    secret BYTEA NOT NULL,
    model TEXT NOT NULL,
    max_tokens INTEGER NOT NULL CHECK (max_tokens BETWEEN 256 AND 8192),
    zdr BOOLEAN NOT NULL DEFAULT true,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ai_credentials_scope_idx ON ai_credentials(workspace_id,user_id);

CREATE TABLE ai_settings (
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    default_credential_id BIGINT REFERENCES ai_credentials(id) ON DELETE SET NULL,
    finance_enabled BOOLEAN NOT NULL DEFAULT false,
    include_balances BOOLEAN NOT NULL DEFAULT false,
    include_goals BOOLEAN NOT NULL DEFAULT false,
    include_obligations BOOLEAN NOT NULL DEFAULT false,
    retention_days INTEGER NOT NULL DEFAULT 30 CHECK (retention_days BETWEEN 1 AND 365),
    daily_limit INTEGER NOT NULL DEFAULT 20 CHECK (daily_limit BETWEEN 1 AND 200),
    PRIMARY KEY (workspace_id,user_id)
);

CREATE TABLE ai_conversations (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    month TEXT NOT NULL,
    revision INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ai_conversations_scope_idx ON ai_conversations(workspace_id,user_id,updated_at DESC);

CREATE TABLE ai_messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('user','assistant')),
    content TEXT NOT NULL,
    context JSONB,
    model TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ai_messages_conversation_idx ON ai_messages(conversation_id,id);

CREATE TABLE ai_runs (
    request_id UUID PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id BIGINT REFERENCES ai_conversations(id) ON DELETE SET NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','completed','failed')),
    input_hash TEXT NOT NULL,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ai_runs_scope_idx ON ai_runs(workspace_id,user_id,created_at DESC);

-- +goose Down
DROP TABLE ai_runs;
DROP TABLE ai_messages;
DROP TABLE ai_conversations;
DROP TABLE ai_settings;
DROP TABLE ai_credentials;
