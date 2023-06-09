CREATE TABLE IF NOT EXISTS requests (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    title text NOT NULL,
    publisher text NOT NULL DEFAULT '',
    isbn varchar(17) NOT NULL DEFAULT '',
    year integer NOT NULL DEFAULT 0,
    waitlist integer NOT NULL DEFAULT 0,
    expiry timestamp(0) with time zone NOT NULL,
    status text NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    version integer NOT NULL DEFAULT 1
);