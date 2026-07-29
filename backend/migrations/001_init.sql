-- +goose Up

CREATE TYPE transaction_type AS ENUM ('income', 'expense');
CREATE TYPE account_kind AS ENUM ('cash', 'bank', 'ewallet', 'investment', 'property', 'liability');

CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    type transaction_type NOT NULL,
    color VARCHAR(7) NOT NULL DEFAULT '#49685c',
    icon VARCHAR(40) NOT NULL DEFAULT 'circle',
    UNIQUE(name, type)
);

CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    kind account_kind NOT NULL,
    current_balance BIGINT NOT NULL DEFAULT 0,
    is_emergency_fund BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    type transaction_type NOT NULL,
    category_id BIGINT NOT NULL REFERENCES categories(id),
    account_id BIGINT NOT NULL REFERENCES accounts(id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    description TEXT NOT NULL DEFAULT '',
    occurred_at DATE NOT NULL,
    is_debt_payment BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX transactions_occurred_at_idx ON transactions(occurred_at);
CREATE INDEX transactions_type_idx ON transactions(type);

CREATE TABLE monthly_budgets (
    id BIGSERIAL PRIMARY KEY,
    category_id BIGINT NOT NULL REFERENCES categories(id),
    month DATE NOT NULL,
    planned_amount BIGINT NOT NULL CHECK (planned_amount >= 0),
    UNIQUE(category_id, month)
);

CREATE TABLE emergency_fund_settings (
    id BIGSERIAL PRIMARY KEY,
    monthly_expense BIGINT NOT NULL CHECK (monthly_expense >= 0),
    target_months SMALLINT NOT NULL CHECK (target_months BETWEEN 1 AND 24),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE financial_goals (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    target_amount BIGINT NOT NULL CHECK (target_amount > 0),
    current_amount BIGINT NOT NULL DEFAULT 0 CHECK (current_amount >= 0),
    target_date DATE NOT NULL,
    icon VARCHAR(40) NOT NULL DEFAULT 'target',
    expected_return NUMERIC(5,2) NOT NULL DEFAULT 6,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE investments (
    id BIGSERIAL PRIMARY KEY,
    asset_type TEXT NOT NULL,
    name TEXT NOT NULL,
    platform TEXT NOT NULL DEFAULT '',
    purchase_value BIGINT NOT NULL CHECK (purchase_value >= 0),
    current_value BIGINT NOT NULL CHECK (current_value >= 0),
    target_allocation NUMERIC(5,2) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE pyramid_items (
    id BIGSERIAL PRIMARY KEY,
    priority SMALLINT NOT NULL CHECK (priority BETWEEN 1 AND 7),
    title TEXT NOT NULL,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    tracker_module TEXT NOT NULL DEFAULT ''
);

CREATE TABLE retirement_settings (
    id BIGSERIAL PRIMARY KEY,
    current_age SMALLINT NOT NULL,
    retirement_age SMALLINT NOT NULL,
    monthly_expense BIGINT NOT NULL,
    inflation_rate NUMERIC(5,2) NOT NULL DEFAULT 4,
    expected_return NUMERIC(5,2) NOT NULL DEFAULT 8,
    current_fund BIGINT NOT NULL DEFAULT 0
);

INSERT INTO categories(name,type,color,icon) VALUES
('Gaji','income','#49685c','briefcase'), ('Freelance','income','#7f9d8e','sparkles'),
('Makanan','expense','#e8a65d','utensils'), ('Transportasi','expense','#7894a0','car'),
('Tempat Tinggal','expense','#d77268','home'), ('Tagihan','expense','#9a8bb7','receipt'),
('Belanja','expense','#b4a464','shopping-bag'), ('Hiburan','expense','#638475','party-popper'),
('Cicilan','expense','#af685f','landmark');

INSERT INTO accounts(name,kind,current_balance,is_emergency_fund) VALUES
('BCA Utama','bank',28500000,FALSE), ('Dana Darurat','bank',21000000,TRUE),
('Dompet','cash',1200000,FALSE), ('Portofolio Investasi','investment',62500000,FALSE),
('KPR','liability',-180000000,FALSE);

INSERT INTO emergency_fund_settings(monthly_expense,target_months) VALUES (7000000,6);
INSERT INTO financial_goals(name,target_amount,current_amount,target_date,icon) VALUES
('Liburan ke Jepang',45000000,27000000,'2027-10-01','plane'),
('DP Rumah',200000000,72000000,'2029-01-01','home'),
('Pendidikan',100000000,18000000,'2030-06-01','graduation-cap');

INSERT INTO investments(asset_type,name,platform,purchase_value,current_value,target_allocation) VALUES
('Reksa Dana','Sucorinvest Equity Fund','Bibit',18000000,21600000,35),
('Saham Indonesia','BBCA','Stockbit',17000000,19400000,30),
('Obligasi','SBR013','Bareksa',15000000,15750000,25),
('Emas','Emas Digital','Pegadaian',5500000,5750000,10);

INSERT INTO transactions(type,category_id,account_id,amount,description,occurred_at,is_debt_payment) VALUES
('income',1,1,18000000,'Gaji bulanan',date_trunc('month',CURRENT_DATE)::date + 1,FALSE),
('income',2,1,3500000,'Proyek desain',date_trunc('month',CURRENT_DATE)::date + 6,FALSE),
('expense',3,1,2850000,'Makan & belanja dapur',date_trunc('month',CURRENT_DATE)::date + 3,FALSE),
('expense',4,1,1200000,'Transportasi',date_trunc('month',CURRENT_DATE)::date + 5,FALSE),
('expense',5,1,4000000,'Sewa tempat tinggal',date_trunc('month',CURRENT_DATE)::date + 1,FALSE),
('expense',6,1,1350000,'Listrik, internet, telepon',date_trunc('month',CURRENT_DATE)::date + 8,FALSE),
('expense',9,1,2500000,'Cicilan kendaraan',date_trunc('month',CURRENT_DATE)::date + 10,TRUE),
('expense',8,1,750000,'Nonton dan kopi',date_trunc('month',CURRENT_DATE)::date + 12,FALSE);

INSERT INTO transactions(type,category_id,account_id,amount,description,occurred_at)
SELECT 'income',1,1, 16000000 + (n * 300000), 'Gaji', (date_trunc('month',CURRENT_DATE) - (n || ' months')::interval + interval '1 day')::date
FROM generate_series(1,5) n;
INSERT INTO transactions(type,category_id,account_id,amount,description,occurred_at)
SELECT 'expense',3,1, 9000000 + (n * 180000), 'Pengeluaran bulanan', (date_trunc('month',CURRENT_DATE) - (n || ' months')::interval + interval '5 days')::date
FROM generate_series(1,5) n;

INSERT INTO pyramid_items(priority,title,is_completed,tracker_module) VALUES
(1,'Punya penghasilan tetap dalam 3 bulan terakhir',TRUE,'cashflow'),
(2,'Pemasukan lebih besar dari pengeluaran',TRUE,'cashflow'),
(3,'Dana darurat 6 bulan pengeluaran tersedia',FALSE,'emergency-fund'),
(4,'Proteksi kesehatan dan jiwa tersedia',FALSE,'protection'),
(5,'Tujuan keuangan sudah direncanakan',TRUE,'goals'),
(6,'Investasi rutin setiap bulan',TRUE,'investments'),
(7,'Dana pensiun berada di jalur yang tepat',FALSE,'retirement');

-- +goose Down

DROP TABLE IF EXISTS retirement_settings;
DROP TABLE IF EXISTS pyramid_items;
DROP TABLE IF EXISTS investments;
DROP TABLE IF EXISTS financial_goals;
DROP TABLE IF EXISTS emergency_fund_settings;
DROP TABLE IF EXISTS monthly_budgets;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS categories;
DROP TYPE IF EXISTS account_kind;
DROP TYPE IF EXISTS transaction_type;
