-- +goose Up
-- Ребаланс под реалии eFootball mobile: 5 голов за матч — обычное дело,
-- планки поднимаем, чтобы награды оставались ценными.
UPDATE achievements SET icon = '🎪',
  name_uz = 'Gol shousi: bitta o''yinda 8+',
  name_ru = 'Голевое шоу: 8+ в одном матче',
  name_tg = 'Шоуи голҳо: 8+ дар як бозӣ'
WHERE code = 'poker_5';

UPDATE achievements SET
  name_uz = 'Triller: 10+ gol, farq ≤2',
  name_ru = 'Триллер: перестрелка 10+ голов',
  name_tg = 'Триллер: 10+ гол, фарқ ≤2'
WHERE code = 'thriller_8';

-- +goose Down
UPDATE achievements SET icon = '🖐',
  name_uz = '5+ gol bir o''yinda', name_ru = '5+ голов в одном матче', name_tg = '5+ гол дар як бозӣ'
WHERE code = 'poker_5';
UPDATE achievements SET
  name_uz = 'Triller g''alabasi', name_ru = 'Триллер: победа 8+ голов', name_tg = 'Триллер: ғалаба 8+ гол'
WHERE code = 'thriller_8';
