CREATE TABLE IF NOT EXISTS books_categories (
    book_id bigint NOT NULL REFERENCES books ON DELETE CASCADE,
    category_id bigint NOT NULL REFERENCES categories ON DELETE CASCADE,
    datetime timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    PRIMARY KEY (book_id, category_id)
);