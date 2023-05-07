CREATE TABLE IF NOT EXISTS book (
    id bigserial PRIMARY KEY,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    title text NOT NULL,
    description text,
    author text[],
    category text,
    publisher text,
    language text,
    series text,
    volume integer,
    edition integer,
    year integer,
    page_count integer,
    isbn_10 text,
    isbn_13 text,
    cover_url text,
    s3_file_key text NOT NULL,
    additional_info jsonb NOT NULL,
    version integer NOT NULL DEFAULT 1
);

ALTER TABLE book ADD CONSTRAINT description_length_check CHECK (LENGTH(description) BETWEEN 1 AND 2000);
ALTER TABLE book ADD CONSTRAINT author_length_check CHECK (ARRAY_LENGTH(author, 1) BETWEEN 1 AND 5);
ALTER TABLE book ADD CONSTRAINT book_volume_check CHECK (volume >= 0);
ALTER TABLE book ADD CONSTRAINT book_edition_check CHECK (edition >= 0);
ALTER TABLE book ADD CONSTRAINT book_year_check CHECK (year BETWEEN 1800 AND date_part('year', now()));
ALTER TABLE book ADD CONSTRAINT book_pagecount_check CHECK (page_count >= 0);
ALTER TABLE book ADD CONSTRAINT isbn10_length_check CHECK (LENGTH(isbn_10) BETWEEN 1 AND 10);
ALTER TABLE book ADD CONSTRAINT isbn13_length_check CHECK (LENGTH(isbn_13) BETWEEN 1 AND 13);