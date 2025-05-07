CREATE TABLE IF NOT EXISTS public.wallets
(
    username text COLLATE pg_catalog."default" NOT NULL,
    password_hash text COLLATE pg_catalog."default" NOT NULL,
    email text COLLATE pg_catalog."default" NOT NULL,
    balance_usd numeric NOT NULL DEFAULT 0,
    balance_rub numeric NOT NULL DEFAULT 0,
    balance_eur numeric NOT NULL DEFAULT 0,
    CONSTRAINT key_primary PRIMARY KEY (username),
    CONSTRAINT unique_credentials UNIQUE (username, email),
    CONSTRAINT negative_balance CHECK (balance_usd >= 0::numeric AND balance_rub >= 0::numeric AND balance_eur >= 0::numeric) NOT VALID
)