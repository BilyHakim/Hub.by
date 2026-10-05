-- +goose Up
CREATE TABLE maintenance_categories (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, name),
    UNIQUE (id, workspace_id)
);

CREATE TABLE maintenance_items (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    category_id BIGINT,
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
    brand TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    purchase_date DATE,
    start_usage_date DATE,
    purchase_price NUMERIC(18,2) CHECK (purchase_price >= 0),
    serial_number TEXT NOT NULL DEFAULT '',
    warranty_expiry DATE,
    location TEXT NOT NULL DEFAULT '',
    image_url TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    current_usage NUMERIC(18,2) CHECK (current_usage >= 0),
    usage_unit TEXT NOT NULL DEFAULT 'km' CHECK (usage_unit IN ('km','hours','cycles')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, workspace_id),
    FOREIGN KEY (category_id, workspace_id) REFERENCES maintenance_categories(id, workspace_id)
);

CREATE TABLE maintenance_rules (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    item_id BIGINT NOT NULL,
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
    kind TEXT NOT NULL CHECK (kind IN ('maintenance','replacement')),
    interval_value INTEGER CHECK (interval_value BETWEEN 1 AND 10000),
    interval_unit TEXT CHECK (interval_unit IN ('days','weeks','months','years')),
    usage_interval NUMERIC(18,2) CHECK (usage_interval > 0),
    start_date DATE NOT NULL,
    start_usage NUMERIC(18,2) NOT NULL DEFAULT 0 CHECK (start_usage >= 0),
    reminder_days INTEGER NOT NULL DEFAULT 14 CHECK (reminder_days BETWEEN 0 AND 365),
    reminder_usage NUMERIC(18,2) NOT NULL DEFAULT 0 CHECK (reminder_usage >= 0),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((interval_value IS NULL) = (interval_unit IS NULL)),
    CHECK (interval_value IS NOT NULL OR usage_interval IS NOT NULL),
    UNIQUE (id, item_id, workspace_id),
    FOREIGN KEY (item_id, workspace_id) REFERENCES maintenance_items(id, workspace_id) ON DELETE CASCADE
);

CREATE TABLE maintenance_histories (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    item_id BIGINT NOT NULL,
    rule_id BIGINT NOT NULL,
    rule_name TEXT NOT NULL,
    completed_at DATE NOT NULL,
    cost NUMERIC(18,2) CHECK (cost >= 0),
    vendor TEXT NOT NULL DEFAULT '',
    usage_value NUMERIC(18,2) CHECK (usage_value >= 0),
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (rule_id, item_id, workspace_id) REFERENCES maintenance_rules(id, item_id, workspace_id) ON DELETE CASCADE
);
CREATE INDEX maintenance_items_workspace_idx ON maintenance_items(workspace_id, updated_at DESC);
CREATE INDEX maintenance_rules_item_idx ON maintenance_rules(workspace_id, item_id);
CREATE INDEX maintenance_histories_rule_date_idx ON maintenance_histories(rule_id, completed_at DESC, id DESC);
CREATE INDEX maintenance_histories_workspace_date_idx ON maintenance_histories(workspace_id, completed_at DESC, id DESC);

-- +goose Down
DROP TABLE maintenance_histories;
DROP TABLE maintenance_rules;
DROP TABLE maintenance_items;
DROP TABLE maintenance_categories;
