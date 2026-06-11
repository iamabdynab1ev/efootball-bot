-- +goose Up
CREATE TABLE achievements (
  id   SERIAL PRIMARY KEY,
  code VARCHAR(32) UNIQUE NOT NULL,
  icon VARCHAR(8) NOT NULL,
  name_uz TEXT NOT NULL,
  name_ru TEXT NOT NULL,
  name_tg TEXT NOT NULL
);

CREATE TABLE user_achievements (
  id             BIGSERIAL PRIMARY KEY,
  user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  achievement_id INT    NOT NULL REFERENCES achievements(id),
  league_id      BIGINT REFERENCES leagues(id) ON DELETE SET NULL,
  earned_at      TIMESTAMPTZ DEFAULT NOW()
);
-- UNIQUE-ограничение таблицы не может содержать выражение — только
-- уникальный индекс. NULL league_id сворачивается в -1, чтобы глобальная
-- ачивка не выдавалась дважды.
CREATE UNIQUE INDEX uq_user_achievements
  ON user_achievements(user_id, achievement_id, COALESCE(league_id, -1));
CREATE INDEX idx_user_achievements_user ON user_achievements(user_id);

INSERT INTO achievements (code, icon, name_uz, name_ru, name_tg) VALUES
  ('first_win',     '🥇', 'Birinchi g''alaba',      'Первая победа',          'Ғалабаи аввал'),
  ('streak_3',      '🔥', '3 g''alaba ketma-ket',   '3 победы подряд',        '3 ғалаба паи ҳам'),
  ('streak_5',      '💥', '5 g''alaba ketma-ket',   '5 побед подряд',         '5 ғалаба паи ҳам'),
  ('streak_10',     '👑', '10 g''alaba ketma-ket',  '10 побед подряд',        '10 ғалаба паи ҳам'),
  ('hat_trick',     '⚽', 'Hat-trick',               'Хет-трик',               'Ҳет-трик'),
  ('scorer_10',     '👟', '10 gol',                  '10 голов за сезон',      '10 гол дар мавсим'),
  ('clean_sheet_5', '🧤', '5 darvoza xatarsiz',      '5 сухих матчей подряд',  '5 дарвозаи покиза'),
  ('veteran',       '🎖️', '50 o''yin',              '50 матчей',              '50 бозӣ'),
  ('league_champion','🏆','Chempion',                'Чемпион лиги',           'Чемпиони лига');

-- +goose Down
DROP TABLE user_achievements;
DROP TABLE achievements;
