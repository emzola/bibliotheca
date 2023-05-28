CREATE TABLE IF NOT EXISTS booklists (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    books jsonb NOT NULL DEFAULT '{}',
    private boolean NOT NULL DEFAULT FALSE,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    version integer NOT NULL DEFAULT 1
);