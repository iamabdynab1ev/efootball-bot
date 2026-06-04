-- +goose Up
CREATE TABLE round_deadlines (
  id BIGSERIAL PRIMARY KEY,
  league_id BIGINT NOT NULL REFERENCES leagues(id) ON DELETE CASCADE,
  round SMALLINT NOT NULL,
  deadline TIMESTAMPTZ NOT NULL,
  reminder_24h_sent BOOLEAN DEFAULT FALSE,
  reminder_1h_sent  BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(league_id, round)
);
CREATE INDEX idx_round_deadlines_reminder ON round_deadlines(deadline)
  WHERE reminder_24h_sent = FALSE OR reminder_1h_sent = FALSE;

-- +goose Down
DROP TABLE round_deadlines;
