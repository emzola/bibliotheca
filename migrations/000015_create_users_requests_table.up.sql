CREATE TABLE IF NOT EXISTS users_requests (
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    request_id bigint NOT NULL REFERENCES requests ON DELETE CASCADE,
    datetime timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    expiry timestamp(0) with time zone NOT NULL,
    PRIMARY KEY (user_id, request_id)
);