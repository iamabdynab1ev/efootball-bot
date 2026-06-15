-- +goose Up

-- Узлы сетки двойной элиминации. В отличие от bracket_slots (одиночная
-- элиминация по стадиям), здесь хранится граф с маршрутизацией проигравших:
-- каждый узел знает, откуда берутся его участники (home_src/away_src):
--   'seed:N' — начальный сид N
--   'win:K'  — победитель узла node_key=K
--   'lose:K' — проигравший узла node_key=K
CREATE TABLE IF NOT EXISTS de_nodes (
    id             BIGSERIAL PRIMARY KEY,
    league_id      BIGINT NOT NULL REFERENCES leagues(id) ON DELETE CASCADE,
    node_key       INT    NOT NULL,            -- стабильный id узла в пределах лиги
    bracket        TEXT   NOT NULL,            -- 'de_w' | 'de_l' | 'de_gf'
    round          INT    NOT NULL,            -- 1-based раунд внутри своей сетки
    ord            INT    NOT NULL,            -- 1-based позиция в раунде
    is_reset       BOOLEAN NOT NULL DEFAULT FALSE, -- второй гранд-финал (bracket reset)
    home_user_id   BIGINT REFERENCES users(id),
    away_user_id   BIGINT REFERENCES users(id),
    home_src       TEXT,
    away_src       TEXT,
    match_id       BIGINT REFERENCES matches(id),
    winner_user_id BIGINT REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(league_id, node_key)
);

CREATE INDEX IF NOT EXISTS idx_de_nodes_league   ON de_nodes(league_id);
CREATE INDEX IF NOT EXISTS idx_de_nodes_match    ON de_nodes(match_id);
-- Быстрый поиск потребителей результата узла при продвижении.
CREATE INDEX IF NOT EXISTS idx_de_nodes_home_src ON de_nodes(league_id, home_src);
CREATE INDEX IF NOT EXISTS idx_de_nodes_away_src ON de_nodes(league_id, away_src);

-- +goose Down
DROP TABLE IF EXISTS de_nodes;
