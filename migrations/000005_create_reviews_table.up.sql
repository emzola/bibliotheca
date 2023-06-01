CREATE TABLE IF NOT EXISTS reviews (
    id bigserial PRIMARY KEY,
    book_id bigint NOT NULL REFERENCES books ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    rating integer NOT NULL,
    comment text NOT NULL DEFAULT '',
    version integer NOT NULL DEFAULT 1
);

ALTER TABLE reviews ADD CONSTRAINT reviews_rating_check CHECK (rating BETWEEN 1 AND 5);