-- +goose Up
-- +goose StatementBegin
-- Журнал действий (аудит): кто, когда и что сделал. Источник истины для
-- админ-мониторинга и разбора споров. actor/target → SET NULL при удалении
-- пользователя, чтобы запись о действии не пропадала вместе с аккаунтом.
CREATE TABLE IF NOT EXISTS audit_log (
    id          BIGSERIAL PRIMARY KEY,
    actor_id    BIGINT REFERENCES users(id)   ON DELETE SET NULL,
    action      TEXT   NOT NULL,
    entity_type TEXT,
    entity_id   BIGINT,
    league_id   BIGINT REFERENCES leagues(id) ON DELETE SET NULL,
    target_id   BIGINT REFERENCES users(id)   ON DELETE SET NULL,
    metadata    JSONB,
    ip          TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы под основные выборки: лента (id DESC), по актору, по цели, по лиге.
-- Покрывают «история действий пользователя» и «события турнира» без seq-scan.
CREATE INDEX IF NOT EXISTS idx_audit_id      ON audit_log (id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_actor   ON audit_log (actor_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_target  ON audit_log (target_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_league  ON audit_log (league_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action  ON audit_log (action, id DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_log;
-- +goose StatementEnd
