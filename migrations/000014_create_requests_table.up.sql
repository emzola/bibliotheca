CREATE TABLE IF NOT EXISTS requests (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    title text NOT NULL,
    author text[] NOT NULL DEFAULT '{}',
    publisher text NOT NULL DEFAULT '',
    isbn varchar(17) NOT NULL DEFAULT '',
    year integer NOT NULL DEFAULT 0,
    language text NOT NULL DEFAULT '',
    waitlist integer NOT NULL DEFAULT 0,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
);