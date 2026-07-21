-- +goose Up
-- Закрытие сезона с церемонией: момент закрытия — для баннера «смотреть
-- церемонию» на главной и сортировки архива в Зале Славы.
-- Идемпотентно: обязана примениться и на исторической прод-схеме.
ALTER TABLE seasons ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ;
UPDATE seasons SET closed_at = updated_at WHERE status = 'finished' AND closed_at IS NULL;

-- Возможные исторические дубли сезонных наград (league_id IS NULL) убираем
-- перед созданием уникального индекса — оставляем самую свежую запись.
DELETE FROM season_awards sa USING season_awards sa2
WHERE sa.league_id IS NULL AND sa2.league_id IS NULL
  AND sa.season_id = sa2.season_id AND sa.award_type = sa2.award_type
  AND sa.id < sa2.id;

-- Сезонные номинации (league_id IS NULL) — по одной на сезон и тип:
-- повторное закрытие сезона идемпотентно обновляет, а не дублирует.
CREATE UNIQUE INDEX IF NOT EXISTS uq_season_awards_season_scope
  ON season_awards (season_id, award_type) WHERE league_id IS NULL;

-- +goose Down
DROP INDEX IF EXISTS uq_season_awards_season_scope;
ALTER TABLE seasons DROP COLUMN IF EXISTS closed_at;
