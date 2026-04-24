-- +goose Up
-- +goose StatementBegin
CREATE TABLE admins (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS admins;
-- +goose StatementEnd


