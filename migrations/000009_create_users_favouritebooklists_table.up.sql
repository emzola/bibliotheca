CREATE TABLE IF NOT EXISTS users_favouritebooklists (
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    booklist_id bigint NOT NULL REFERENCES booklists ON DELETE CASCADE,
    datetime timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, booklist_id)
);