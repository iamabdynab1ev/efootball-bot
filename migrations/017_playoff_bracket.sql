-- +goose Up

-- Добавляем этап к матчам (league / qf / sf / final / r16 / r32)
ALTER TABLE matches ADD COLUMN IF NOT EXISTS stage TEXT NOT NULL DEFAULT 'league';
ALTER TABLE matches ADD COLUMN IF NOT EXISTS bracket_slot INT;

-- Сетка плей-офф: отслеживаем кто в каком слоте каждого этапа
CREATE TABLE IF NOT EXISTS bracket_slots (
    id           SERIAL PRIMARY KEY,
    league_id    INT  NOT NULL REFERENCES leagues(id) ON DELETE CASCADE,
    stage        TEXT NOT NULL,  -- 'r32','r16','qf','sf','final'
    slot         INT  NOT NULL,  -- 1-based position in stage
    home_user_id INT  REFERENCES users(id),
    away_user_id INT  REFERENCES users(id),
    match_id     INT  REFERENCES matches(id),
    winner_user_id INT REFERENCES users(id),
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(league_id, stage, slot)
);

CREATE INDEX IF NOT EXISTS idx_bracket_league_stage ON bracket_slots(league_id, stage);
CREATE INDEX IF NOT EXISTS idx_matches_stage ON matches(league_id, stage);

-- +goose Down
DROP TABLE IF EXISTS bracket_slots;
ALTER TABLE matches DROP COLUMN IF EXISTS stage;
ALTER TABLE matches DROP COLUMN IF EXISTS bracket_slot;
