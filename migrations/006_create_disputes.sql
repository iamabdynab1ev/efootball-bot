-- +goose Up
-- +goose StatementBegin
CREATE TABLE disputes (
    id           BIGSERIAL PRIMARY KEY,
    match_id     BIGINT    NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    reported_by  BIGINT    NOT NULL REFERENCES users(id),         -- кто открыл спор (гость)

    -- Что каждый заявил
    home_claimed SMALLINT  NOT NULL,
    away_claimed SMALLINT  NOT NULL,

    -- Решение админа
    resolved        BOOLEAN   NOT NULL DEFAULT FALSE,
    resolved_by     BIGINT    REFERENCES users(id),               -- telegram_id админа
    admin_home_goals SMALLINT,
    admin_away_goals SMALLINT,
    admin_note      TEXT,

    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    resolved_at  TIMESTAMP
);

CREATE INDEX idx_disputes_match_id    ON disputes(match_id);
CREATE INDEX idx_disputes_resolved    ON disputes(resolved);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS disputes;
-- +goose StatementEnd
