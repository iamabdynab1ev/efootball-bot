-- +goose Up
-- Новые достижения: матчевые события, карьерные вехи, титулы.
INSERT INTO achievements (code, icon, name_uz, name_ru, name_tg) VALUES
  ('poker_5',      '🖐', '5+ gol bir o''yinda',     '5+ голов в одном матче',   '5+ гол дар як бозӣ'),
  ('thriller_8',   '🧨', 'Triller g''alabasi',      'Триллер: победа 8+ голов', 'Триллер: ғалаба 8+ гол'),
  ('veteran_100',  '🏅', '100 o''yin',              '100 матчей',               '100 бозӣ'),
  ('veteran_200',  '🏛', '200 o''yin',              '200 матчей',               '200 бозӣ'),
  ('goals_100',    '💯', '100 gol (karyera)',       'Клуб 100 голов',           'Клуби 100 гол'),
  ('goals_250',    '🚀', '250 gol (karyera)',       'Клуб 250 голов',           'Клуби 250 гол'),
  ('goals_500',    '🌋', '500 gol (karyera)',       'Клуб 500 голов',           'Клуби 500 гол'),
  ('elo_1200',     '📈', 'Reyting 1200',            'Рейтинг 1200',             'Рейтинги 1200'),
  ('elo_1300',     '🚁', 'Reyting 1300',            'Рейтинг 1300',             'Рейтинги 1300'),
  ('champ_2',      '👑', '2 karra chempion',        'Двукратный чемпион',       'Дукарата чемпион'),
  ('champ_3',      '💎', '3 karra chempion',        'Трёхкратный чемпион',      'Секарата чемпион'),
  ('champ_5',      '🌟', '5 karra chempion',        'Легенда — 5 титулов',      'Афсона — 5 унвон')
ON CONFLICT (code) DO NOTHING;

-- +goose Down
DELETE FROM achievements WHERE code IN
  ('poker_5','thriller_8','veteran_100','veteran_200','goals_100','goals_250','goals_500',
   'elo_1200','elo_1300','champ_2','champ_3','champ_5');
