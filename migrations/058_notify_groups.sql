-- +goose Up
-- Мульти-группы уведомлений: бот подключается к нескольким группам (Telegram
-- и/или WhatsApp), а каждая лига может слать свои новости в конкретную группу
-- («маршрут лига → группа»). Если у лиги маршрут не задан (notify_group_id NULL),
-- новости уходят во все включённые группы — как было до этой фичи.
CREATE TABLE notify_groups (
  id         BIGSERIAL PRIMARY KEY,
  channel    TEXT NOT NULL CHECK (channel IN ('telegram','whatsapp')),
  chat_id    TEXT NOT NULL,                 -- TG chat_id (как текст) или WA jid
  title      TEXT NOT NULL DEFAULT '',
  enabled    BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (channel, chat_id)
);

ALTER TABLE leagues
  ADD COLUMN IF NOT EXISTS notify_group_id BIGINT REFERENCES notify_groups(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE leagues DROP COLUMN IF EXISTS notify_group_id;
DROP TABLE IF EXISTS notify_groups;
