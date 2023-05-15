CREATE EXTENSION btree_gin;
CREATE INDEX IF NOT EXISTS books_title_idx ON books USING GIN (to_tsvector('simple', title));
CREATE INDEX IF NOT EXISTS books_author_idx ON books USING GIN (author);
CREATE INDEX IF NOT EXISTS books_isbn10_idx ON books USING GIN (isbn_10);
CREATE INDEX IF NOT EXISTS books_isbn13_idx ON books USING GIN (isbn_13);
CREATE INDEX IF NOT EXISTS books_publisher_idx ON books USING GIN (to_tsvector('simple', publisher));
CREATE INDEX IF NOT EXISTS books_year_idx ON books USING GIN (year);
CREATE INDEX IF NOT EXISTS books_language_idx ON books USING GIN (language);
CREATE INDEX IF NOT EXISTS books_extension_idx ON books USING GIN (extension);
