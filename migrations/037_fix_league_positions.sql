-- +goose Up
-- +goose StatementBegin
UPDATE league_members lm
SET position = sub.pos,
    updated_at = NOW()
FROM (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY league_id
               ORDER BY points DESC,
                        (goals_for - goals_against) DESC,
                        goals_for DESC,
                        id ASC
           ) AS pos
    FROM league_members
    WHERE status = 'approved'
) sub
WHERE lm.id = sub.id;

CREATE UNIQUE INDEX IF NOT EXISTS uq_league_members_league_position
    ON league_members (league_id, position)
    WHERE status = 'approved' AND position IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS uq_league_members_league_position;
-- +goose StatementEnd
