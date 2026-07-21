-- +goose Up
-- Дедлайны 2.0: поддержка стадий плей-офф и автоматическое исполнение.
-- stage='' + round>0 — тур группового/лигового этапа; stage='qf'... + round=0 —
-- стадия плей-офф. processed_at — автоматика уже выставила технические
-- результаты (идемпотентность тикера).
--
-- Все шаги идемпотентны (IF EXISTS / IF NOT EXISTS): миграция обязана
-- примениться на проде даже при частично отличающейся исторической схеме.
ALTER TABLE round_deadlines ADD COLUMN IF NOT EXISTS stage TEXT NOT NULL DEFAULT '';
ALTER TABLE round_deadlines ADD COLUMN IF NOT EXISTS processed_at TIMESTAMPTZ;

-- Старая уникальность (league_id, round): имя могло отличаться в исторической
-- базе — снимаем и как constraint, и как отдельный индекс.
ALTER TABLE round_deadlines DROP CONSTRAINT IF EXISTS round_deadlines_league_id_round_key;
DROP INDEX IF EXISTS round_deadlines_league_id_round_key;

CREATE UNIQUE INDEX IF NOT EXISTS uq_round_deadlines_scope ON round_deadlines (league_id, round, stage);
CREATE INDEX IF NOT EXISTS idx_round_deadlines_due ON round_deadlines (deadline) WHERE processed_at IS NULL;

-- Старые дедлайны, чей срок уже прошёл, считаем отработанными: не выставлять
-- задним числом технические результаты по турам, сыгранным до этого релиза.
UPDATE round_deadlines SET processed_at = NOW() WHERE deadline <= NOW() AND processed_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_round_deadlines_due;
DROP INDEX IF EXISTS uq_round_deadlines_scope;
ALTER TABLE round_deadlines DROP COLUMN IF EXISTS processed_at;
ALTER TABLE round_deadlines DROP COLUMN IF EXISTS stage;
