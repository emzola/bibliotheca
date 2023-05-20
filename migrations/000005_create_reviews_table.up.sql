CREATE TABLE IF NOT EXISTS reviews (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    rating integer NOT NULL,
    review text NOT NULL,
);