-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT       NOT NULL UNIQUE,   -- уникальный ID телеграма, есть у всех
    display_name VARCHAR(64) NOT NULL,           -- имя которое сам ввёл
    username    VARCHAR(64),                     -- @username если есть (может быть NULL)
    is_banned   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_telegram_id ON users(telegram_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
