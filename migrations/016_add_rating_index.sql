-- +goose Up
CREATE INDEX IF NOT EXISTS idx_users_rating_desc
    ON users(rating DESC)
    WHERE is_banned = false;

CREATE INDEX IF NOT EXISTS idx_matches_history
    ON matches(home_user_id, away_user_id, status, played_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_users_rating_desc;
DROP INDEX IF EXISTS idx_matches_history;
