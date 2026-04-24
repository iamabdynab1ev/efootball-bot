-- +goose Up
-- +goose StatementBegin

UPDATE users SET language = 'uz' WHERE language = '' OR language IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- rollback йўқ

-- +goose StatementEnd