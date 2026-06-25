-- +goose Up
-- Код (CreateLeague) создаёт лигу со статусом 'draft', но в enum его не было
-- (миграция, добавлявшая его, потерялась при консолидации 018–025).
-- На свежей БД создание лиги падало. IF NOT EXISTS — безопасно для прода.
ALTER TYPE league_status ADD VALUE IF NOT EXISTS 'draft';

-- +goose Down
-- Значения enum в PostgreSQL не удаляются.
