CREATE TABLE IF NOT EXISTS comments (
    id bigserial PRIMARY KEY,
    parent_id bigint REFERENCES comments ON DELETE CASCADE,
    booklist_id bigint NOT NULL REFERENCES booklists ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    content text NOT NULL,
    version integer NOT NULL DEFAULT 1
);