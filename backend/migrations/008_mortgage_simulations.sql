-- +goose Up

CREATE TABLE mortgage_simulations (
    workspace_id BIGINT PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    property_price BIGINT NOT NULL DEFAULT 500000000 CHECK (property_price > 0),
    down_payment_percent NUMERIC(5,2) NOT NULL DEFAULT 20 CHECK (down_payment_percent BETWEEN 0 AND 100),
    tenor_years SMALLINT NOT NULL DEFAULT 15 CHECK (tenor_years BETWEEN 1 AND 30),
    fixed_rate NUMERIC(6,3) NOT NULL DEFAULT 5 CHECK (fixed_rate BETWEEN 0 AND 100),
    fixed_years SMALLINT NOT NULL DEFAULT 5 CHECK (fixed_years BETWEEN 0 AND 30),
    floating_rate NUMERIC(6,3) NOT NULL DEFAULT 10 CHECK (floating_rate BETWEEN 0 AND 100),
    monthly_income BIGINT NOT NULL DEFAULT 20000000 CHECK (monthly_income >= 0),
    other_installments BIGINT NOT NULL DEFAULT 0 CHECK (other_installments >= 0),
    other_costs BIGINT NOT NULL DEFAULT 0 CHECK (other_costs >= 0),
    start_date DATE NOT NULL DEFAULT CURRENT_DATE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (fixed_years <= tenor_years)
);

-- +goose Down

DROP TABLE IF EXISTS mortgage_simulations;
