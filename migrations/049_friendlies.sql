-- +goose Up
-- Товарищеские матчи: вызов друга без турнира. Счёт подтверждается соперником
-- и влияет на ELO-рейтинг профиля.
CREATE TABLE friendlies (
  id               BIGSERIAL PRIMARY KEY,
  challenger_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  opponent_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  -- pending → accepted → score_claimed → confirmed; либо declined/cancelled
  status           TEXT NOT NULL DEFAULT 'pending',
  challenger_goals SMALLINT,
  opponent_goals   SMALLINT,
  claimed_by       BIGINT REFERENCES users(id),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_friendlies_challenger ON friendlies(challenger_id, status);
CREATE INDEX idx_friendlies_opponent   ON friendlies(opponent_id, status);
-- Один незавершённый вызов между парой игроков за раз.
CREATE UNIQUE INDEX uq_friendlies_active_pair ON friendlies (
  LEAST(challenger_id, opponent_id), GREATEST(challenger_id, opponent_id)
) WHERE status IN ('pending', 'accepted', 'score_claimed');

-- +goose Down
DROP TABLE friendlies;
