-- +goose Up
-- Дедлайны 2.0: поддержка стадий плей-офф и автоматическое исполнение.
-- stage='' + round>0 — тур группового/лигового этапа; stage='qf'... + round=0 —
-- стадия плей-офф. processed_at — автоматика уже выставила технические
-- результаты (идемпотентность тикера).
ALTER TABLE round_deadlines ADD COLUMN stage TEXT NOT NULL DEFAULT '';
ALTER TABLE round_deadlines ADD COLUMN processed_at TIMESTAMPTZ;
ALTER TABLE round_deadlines DROP CONSTRAINT round_deadlines_league_id_round_key;
CREATE UNIQUE INDEX uq_round_deadlines_scope ON round_deadlines (league_id, round, stage);
CREATE INDEX idx_round_deadlines_due ON round_deadlines (deadline) WHERE processed_at IS NULL;

-- Старые дедлайны, чей срок уже прошёл, считаем отработанными: не выставлять
-- задним числом технические результаты по турам, сыгранным до этого релиза.
UPDATE round_deadlines SET processed_at = NOW() WHERE deadline <= NOW();

-- +goose Down
DROP INDEX IF EXISTS idx_round_deadlines_due;
DROP INDEX IF EXISTS uq_round_deadlines_scope;
ALTER TABLE round_deadlines ADD CONSTRAINT round_deadlines_league_id_round_key UNIQUE (league_id, round);
ALTER TABLE round_deadlines DROP COLUMN processed_at;
ALTER TABLE round_deadlines DROP COLUMN stage;
