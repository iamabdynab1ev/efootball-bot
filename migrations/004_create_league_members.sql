-- +goose Up
-- +goose StatementBegin
CREATE TYPE member_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE league_members (
    id           BIGSERIAL     PRIMARY KEY,
    league_id    BIGINT        NOT NULL REFERENCES leagues(id) ON DELETE CASCADE,
    user_id      BIGINT        NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    status       member_status NOT NULL DEFAULT 'pending',      -- ждёт одобрения админа

    -- Статистика (обновляется после каждого матча)
    points       SMALLINT      NOT NULL DEFAULT 0,
    wins         SMALLINT      NOT NULL DEFAULT 0,
    draws        SMALLINT      NOT NULL DEFAULT 0,
    losses       SMALLINT      NOT NULL DEFAULT 0,
    goals_for    SMALLINT      NOT NULL DEFAULT 0,
    goals_against SMALLINT     NOT NULL DEFAULT 0,
    position     SMALLINT,                                       -- место в таблице (пересчитывается)

    joined_at    TIMESTAMP     NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP     NOT NULL DEFAULT NOW(),

    UNIQUE(league_id, user_id)                                  -- игрок один раз в лиге
);

CREATE INDEX idx_league_members_league_id ON league_members(league_id);
CREATE INDEX idx_league_members_user_id   ON league_members(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS league_members;
DROP TYPE IF EXISTS member_status;
-- +goose StatementEnd
