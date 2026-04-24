-- +goose Up
-- +goose StatementBegin

-- Для GetMemberStats (вызывается при каждом открытии профиля/расписания)
CREATE INDEX IF NOT EXISTS idx_league_members_user_league
ON league_members(user_id, league_id);

-- Для GetUserLeagues (новый метод — профиль и расписание)
CREATE INDEX IF NOT EXISTS idx_league_members_user_status
ON league_members(user_id, status);

-- Для GetAllDisputed и GetPendingForUser
CREATE INDEX IF NOT EXISTS idx_matches_status_updated
ON matches(status, updated_at DESC);

-- Для GetUserSchedule (самый частый запрос)
CREATE INDEX IF NOT EXISTS idx_matches_league_home
ON matches(league_id, home_user_id);

CREATE INDEX IF NOT EXISTS idx_matches_league_away
ON matches(league_id, away_user_id);

-- Для GetUserMatchHistory
CREATE INDEX IF NOT EXISTS idx_matches_confirmed_played
ON matches(status, played_at DESC)
WHERE status = 'confirmed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_league_members_user_league;
DROP INDEX IF EXISTS idx_league_members_user_status;
DROP INDEX IF EXISTS idx_matches_status_updated;
DROP INDEX IF EXISTS idx_matches_league_home;
DROP INDEX IF EXISTS idx_matches_league_away;
DROP INDEX IF EXISTS idx_matches_confirmed_played;

-- +goose StatementEnd