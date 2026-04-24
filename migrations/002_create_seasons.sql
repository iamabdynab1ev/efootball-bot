-- +goose Up
-- +goose StatementBegin
CREATE TYPE season_status AS ENUM ('pending', 'active', 'finished');

CREATE TABLE seasons (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(64)  NOT NULL,            -- "Сезон 1", "Сезон 2"
    status     season_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP    NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS seasons;
DROP TYPE IF EXISTS season_status;
-- +goose StatementEnd
