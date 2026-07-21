-- +goose Up
-- Прогнозы на матчи: игроки ставят счёт на чужие матчи лиги (виртуальные
-- очки, бесплатно). Очки начисляются при подтверждении матча:
-- точный счёт — 5, верная разница мячей — 3, верный исход — 1, мимо — 0.
-- Прогнозы скрыты от других до закрытия матча (честность).
CREATE TABLE IF NOT EXISTS predictions (
    id         BIGSERIAL PRIMARY KEY,
    match_id   BIGINT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    home_goals SMALLINT NOT NULL,
    away_goals SMALLINT NOT NULL,
    points     SMALLINT,            -- NULL до подтверждения матча, потом 0/1/3/5
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (match_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_predictions_match ON predictions (match_id);
CREATE INDEX IF NOT EXISTS idx_predictions_user ON predictions (user_id);

-- +goose Down
DROP TABLE predictions;
