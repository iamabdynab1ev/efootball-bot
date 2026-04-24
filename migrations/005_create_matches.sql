-- +goose Up
-- +goose StatementBegin
CREATE TYPE match_status AS ENUM (
    'scheduled',        -- матч назначен, ещё не сыгран
    'pending_confirm',  -- хозяин ввёл счёт, ждём подтверждения гостя
    'disputed',         -- гость не согласен, ждём повторного ввода хозяина
    'confirmed',        -- оба согласились, очки начислены
    'cancelled'         -- матч отменён админом
);

CREATE TABLE matches (
    id              BIGSERIAL    PRIMARY KEY,
    league_id       BIGINT       NOT NULL REFERENCES leagues(id) ON DELETE CASCADE,
    home_user_id    BIGINT       NOT NULL REFERENCES users(id),
    away_user_id    BIGINT       NOT NULL REFERENCES users(id),
    round           SMALLINT     NOT NULL,                        -- номер тура

    -- Финальный счёт (заполняется после confirmed)
    home_goals      SMALLINT,
    away_goals      SMALLINT,

    -- Что заявил хозяин (может меняться при dispute)
    claimed_home    SMALLINT,
    claimed_away    SMALLINT,

    status          match_status NOT NULL DEFAULT 'scheduled',
    dispute_count   SMALLINT     NOT NULL DEFAULT 0,              -- сколько раз был спор

    played_at       TIMESTAMP,
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),

    CHECK (home_user_id <> away_user_id)
);

CREATE INDEX idx_matches_league_id     ON matches(league_id);
CREATE INDEX idx_matches_home_user_id  ON matches(home_user_id);
CREATE INDEX idx_matches_away_user_id  ON matches(away_user_id);
CREATE INDEX idx_matches_status        ON matches(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS matches;
DROP TYPE IF EXISTS match_status;
-- +goose StatementEnd
