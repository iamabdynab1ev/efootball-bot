-- +goose Up
-- Ремонт зависших сеток плей-офф: при некруглом числе участников (например,
-- 9 из трёх групп) пары, где ОБА игрока попали в стадию по bye, оставались
-- без матча — buildSeededBracket создавал матчи только для первой стадии.
-- Создаём недостающие матчи для всех слотов, где обе стороны известны,
-- матч не создан и победитель ещё не определён.
INSERT INTO matches (league_id, home_user_id, away_user_id, round, stage, bracket_slot)
SELECT bs.league_id, bs.home_user_id, bs.away_user_id, 100 + bs.slot, bs.stage, bs.slot
FROM bracket_slots bs
JOIN leagues l ON l.id = bs.league_id AND l.status = 'active'
WHERE bs.home_user_id IS NOT NULL
  AND bs.away_user_id IS NOT NULL
  AND bs.match_id IS NULL
  AND bs.winner_user_id IS NULL;

-- Привязываем созданные матчи к слотам.
UPDATE bracket_slots bs
SET match_id = m.id
FROM matches m
WHERE m.league_id = bs.league_id
  AND m.stage = bs.stage
  AND m.bracket_slot = bs.slot
  AND bs.match_id IS NULL
  AND bs.home_user_id IS NOT NULL
  AND bs.away_user_id IS NOT NULL;

-- +goose Down
-- Данные-фикс: откат не требуется (созданные матчи остаются валидными).
SELECT 1;
