-- +goose Up
-- +goose StatementBegin
CREATE TYPE league_status AS ENUM ('registration', 'active', 'finished');

CREATE TABLE leagues (
    id          BIGSERIAL     PRIMARY KEY,
    season_id   BIGINT        NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    name        VARCHAR(64)   NOT NULL,           -- "АПЛ", "Ла Лига" и тд
    country     VARCHAR(64),                      -- "Англия", "Испания"
    level       SMALLINT      NOT NULL DEFAULT 1, -- 1=нац.лига, 2=ЛЧ, 3=ЛЕ
    max_players SMALLINT      NOT NULL DEFAULT 20,
    rounds_type VARCHAR(16)   NOT NULL DEFAULT 'double', -- 'single' или 'double' (дом/выезд)
    status      league_status NOT NULL DEFAULT 'registration',
    created_at  TIMESTAMP     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_leagues_season_id ON leagues(season_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS leagues;
DROP TYPE IF EXISTS league_status;
-- +goose StatementEnd
