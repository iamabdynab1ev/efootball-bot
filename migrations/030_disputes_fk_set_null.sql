-- +goose Up
-- При удалении админа ссылка из разрешённых им споров должна обнуляться,
-- а не блокировать удаление.
ALTER TABLE disputes DROP CONSTRAINT IF EXISTS disputes_resolved_by_fkey;
ALTER TABLE disputes
  ADD CONSTRAINT disputes_resolved_by_fkey
  FOREIGN KEY (resolved_by) REFERENCES users(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE disputes DROP CONSTRAINT IF EXISTS disputes_resolved_by_fkey;
ALTER TABLE disputes
  ADD CONSTRAINT disputes_resolved_by_fkey
  FOREIGN KEY (resolved_by) REFERENCES users(id);
