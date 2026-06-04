-- +goose Up
CREATE TABLE season_awards (
  id         BIGSERIAL PRIMARY KEY,
  season_id  BIGINT NOT NULL REFERENCES seasons(id),
  league_id  BIGINT REFERENCES leagues(id) ON DELETE SET NULL,
  award_type VARCHAR(32) NOT NULL,
  user_id    BIGINT NOT NULL REFERENCES users(id),
  value      INTEGER,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_season_awards_season ON season_awards(season_id);

-- +goose Down
DROP TABLE season_awards;
