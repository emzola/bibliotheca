CREATE TABLE IF NOT EXISTS users_downloads (
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    book_id bigint NOT NULL REFERENCES books ON DELETE CASCADE,
    datetime timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, book_id)
);