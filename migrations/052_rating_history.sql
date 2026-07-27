-- +goose Up
-- История ELO-рейтинга: точка после каждого пересчёта (лига и товарищеские).
-- Питает график динамики на профиле игрока.
CREATE TABLE rating_history (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating     INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_rating_history_user ON rating_history (user_id, id);

-- Стартовая точка для существующих игроков — текущий рейтинг.
INSERT INTO rating_history (user_id, rating)
SELECT id, rating FROM users WHERE is_banned = false;

-- +goose Down
DROP TABLE rating_history;
