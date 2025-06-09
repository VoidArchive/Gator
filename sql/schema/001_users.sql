-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY,
    created_at timestamp NOT NULL,
    updated_at timestamp NOT NULL,
    name text NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE users;

