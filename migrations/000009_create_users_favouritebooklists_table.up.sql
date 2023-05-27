CREATE TABLE IF NOT EXISTS users_favouritebooklists (
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    booklist_id bigint NOT NULL REFERENCES booklists ON DELETE CASCADE,
    PRIMARY KEY (user_id, booklist_id)
);