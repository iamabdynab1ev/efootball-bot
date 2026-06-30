// Package testsupport содержит общие помощники для интеграционных тестов.
package testsupport

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// migrateLockKey — произвольный ключ advisory-lock, общий для всех тестовых
// пакетов, чтобы сериализовать применение миграций.
const migrateLockKey = 987654321

// MigrateLocked применяет миграции к тестовой БД под Postgres advisory-lock.
// Без него параллельные тестовые пакеты (`go test ./...` гоняет пакеты
// одновременно) одновременно вызывают goose.Up на ОДНОЙ базе и конфликтуют на
// создании enum-типов (duplicate key pg_type). Лок сериализует: первый
// применяет миграции, остальные после освобождения видят актуальную версию и
// ничего не делают. dir — путь к каталогу миграций относительно пакета теста.
func MigrateLocked(t *testing.T, dsn, dir string) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	// Лок держим на одном соединении (sql.DB — пул, иначе unlock уйдёт в другое).
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("db.Conn: %v", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_lock($1)", migrateLockKey); err != nil {
		t.Fatalf("advisory lock: %v", err)
	}
	defer conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", migrateLockKey)

	goose.SetDialect("postgres")
	if err := goose.Up(db, dir); err != nil {
		t.Fatalf("migrations: %v", err)
	}
}
