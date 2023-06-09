CREATE TABLE IF NOT EXISTS books (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    title text NOT NULL,
    description text NOT NULL DEFAULT '',
    author text[] NOT NULL DEFAULT '{}',
    category text NOT NULL DEFAULT '',
    publisher text NOT NULL DEFAULT '',
    language text NOT NULL DEFAULT '',
    series text NOT NULL DEFAULT '',
    volume integer NOT NULL DEFAULT 0,
    edition text NOT NULL DEFAULT '',
    year integer NOT NULL DEFAULT 0,
    page_count integer NOT NULL DEFAULT 0,
    isbn_10 varchar(13) NOT NULL DEFAULT '',
    isbn_13 varchar(17) NOT NULL DEFAULT '',
    cover_path text NOT NULL DEFAULT '',
    s3_file_key text NOT NULL,
    fname text NOT NULL,
    extension text NOT NULL, 
    size integer NOT NULL,
    popularity numeric(2, 1) NOT NULL DEFAULT 0,
    version integer NOT NULL DEFAULT 1
);