CREATE TABLE IF NOT EXISTS booklists_books (
    booklist_id bigint NOT NULL REFERENCES booklists ON DELETE CASCADE,
    book_id bigint NOT NULL REFERENCES books ON DELETE CASCADE,
    datetime timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    PRIMARY KEY (booklist_id, book_id)
);