CREATE INDEX IF NOT EXISTS requests_title_idx ON requests USING GIN (to_tsvector('simple', title));
CREATE INDEX IF NOT EXISTS requests_isbn_idx ON requests USING GIN (to_tsvector('simple', isbn));
CREATE INDEX IF NOT EXISTS requests_pub_idx ON requests USING GIN (to_tsvector('simple', publisher));
CREATE INDEX IF NOT EXISTS requests_status_idx ON requests USING GIN (status);