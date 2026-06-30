-- +goose Up
-- +goose StatementBegin
-- Миграция 037 создала UNIQUE INDEX (league_id, position) исходя из того, что
-- позиция уникальна в пределах лиги. Это верно только для лиг БЕЗ групп.
-- В турнире с группами RecalculateTable считает места ВНУТРИ каждой группы
-- (PARTITION BY group_name → group A: 1..N, group B: 1..N), поэтому значения
-- позиций повторяются внутри лиги и нарушают этот индекс — пересчёт таблицы
-- падает при подтверждении матча. Корректность мест гарантирует ROW_NUMBER()
-- в RecalculateTable; отдельный UNIQUE-констрейнт на (league_id, position) для
-- групповых турниров неверен по определению, поэтому удаляем его.
DROP INDEX IF EXISTS uq_league_members_league_position;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS uq_league_members_league_position
    ON league_members (league_id, position)
    WHERE status = 'approved' AND position IS NOT NULL;
-- +goose StatementEnd
