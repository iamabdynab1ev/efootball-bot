-- +goose Up
-- +goose StatementBegin
-- Внутри-приложенческие уведомления. Персистятся (в отличие от живого SSE),
-- чтобы пользователь не терял события офлайн: при заходе/реконнекте догружает
-- непрочитанные. Telegram/web-push остаются параллельными каналами.
CREATE TABLE IF NOT EXISTS notifications (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       TEXT   NOT NULL,
    title      TEXT   NOT NULL,
    body       TEXT,
    link       TEXT,
    read       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Лента пользователя (id DESC) и быстрый счётчик непрочитанных (partial index).
CREATE INDEX IF NOT EXISTS idx_notif_user   ON notifications (user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_notif_unread ON notifications (user_id) WHERE read = FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notifications;
-- +goose StatementEnd
