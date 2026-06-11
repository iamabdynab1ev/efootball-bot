-- +goose Up
-- Восстановление потерянных миграций 18–25: они были применены к живой БД
-- 2–3 июня (goose_db_version), но их файлы не попали в репозиторий.
-- На живой БД версия 22 уже числится применённой — goose пропустит этот файл;
-- на свежей БД он создаст все колонки. IF NOT EXISTS — страховка на случай
-- частично восстановленных копий.
ALTER TABLE leagues
  ADD COLUMN IF NOT EXISTS registration_deadline TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS num_groups      SMALLINT NOT NULL DEFAULT 4,
  ADD COLUMN IF NOT EXISTS group_advance   SMALLINT NOT NULL DEFAULT 2,
  ADD COLUMN IF NOT EXISTS best_runners_up SMALLINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS current_round   SMALLINT NOT NULL DEFAULT 0;

ALTER TABLE league_members
  ADD COLUMN IF NOT EXISTS group_name    VARCHAR(4),
  ADD COLUMN IF NOT EXISTS division_name VARCHAR(10);

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS favorite_club TEXT;

-- Тоже из потерянных миграций: индексы под основные выборки и
-- уникальность пары в туре (защита от дублей расписания).
CREATE INDEX IF NOT EXISTS idx_league_members_group
  ON league_members(league_id, group_name) WHERE group_name IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_league_members_division
  ON league_members(league_id, division_name) WHERE division_name IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_league_members_goals
  ON league_members(league_id, goals_for DESC) WHERE status = 'approved';
CREATE INDEX IF NOT EXISTS idx_matches_bracket_slot
  ON matches(league_id, stage, bracket_slot) WHERE bracket_slot IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_matches_home_status
  ON matches(home_user_id, status)
  WHERE status IN ('pending_confirm', 'disputed', 'scheduled');
CREATE INDEX IF NOT EXISTS idx_matches_away_status
  ON matches(away_user_id, status)
  WHERE status IN ('pending_confirm', 'disputed', 'scheduled');
CREATE UNIQUE INDEX IF NOT EXISTS idx_matches_unique_pair_round
  ON matches(league_id, home_user_id, away_user_id, round) WHERE stage = 'league';

-- +goose Down
DROP INDEX IF EXISTS idx_matches_unique_pair_round;
DROP INDEX IF EXISTS idx_matches_away_status;
DROP INDEX IF EXISTS idx_matches_home_status;
DROP INDEX IF EXISTS idx_matches_bracket_slot;
DROP INDEX IF EXISTS idx_league_members_goals;
DROP INDEX IF EXISTS idx_league_members_division;
DROP INDEX IF EXISTS idx_league_members_group;
ALTER TABLE users DROP COLUMN IF EXISTS favorite_club;
ALTER TABLE league_members
  DROP COLUMN IF EXISTS division_name,
  DROP COLUMN IF EXISTS group_name;
ALTER TABLE leagues
  DROP COLUMN IF EXISTS current_round,
  DROP COLUMN IF EXISTS best_runners_up,
  DROP COLUMN IF EXISTS group_advance,
  DROP COLUMN IF EXISTS num_groups,
  DROP COLUMN IF EXISTS registration_deadline;
