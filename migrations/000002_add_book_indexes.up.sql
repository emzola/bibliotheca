CREATE EXTENSION btree_gin;
CREATE INDEX IF NOT EXISTS book_title_idx ON book USING GIN (to_tsvector('simple', title));
CREATE INDEX IF NOT EXISTS book_author_idx ON book USING GIN (author);
CREATE INDEX IF NOT EXISTS book_isbn10_idx ON book USING GIN (isbn_10);
CREATE INDEX IF NOT EXISTS book_isbn13_idx ON book USING GIN (isbn_13);
CREATE INDEX IF NOT EXISTS book_publisher_idx ON book USING GIN (to_tsvector('simple', publisher));
CREATE INDEX IF NOT EXISTS book_year_idx ON book USING GIN (year);
CREATE INDEX IF NOT EXISTS book_language_idx ON book USING GIN (language);
CREATE INDEX IF NOT EXISTS book_extension_idx ON book USING GIN (extension);
