// Package storage — загрузка медиа (голосовые, фото) в объектное хранилище
// Cloudflare R2 (S3-совместимое). Если переменные окружения не заданы, клиент
// не создаётся и функция медиа просто отключена (Enabled()==false).
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type R2 struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

// NewR2FromEnv читает конфиг из окружения. Возвращает (nil, nil), если R2 не
// настроен (фича отключена, приложение работает как обычно).
func NewR2FromEnv() (*R2, error) {
	acc := os.Getenv("R2_ACCOUNT_ID")
	key := os.Getenv("R2_ACCESS_KEY_ID")
	secret := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucket := os.Getenv("R2_BUCKET")
	pub := strings.TrimRight(os.Getenv("R2_PUBLIC_URL"), "/")
	if acc == "" || key == "" || secret == "" || bucket == "" || pub == "" {
		return nil, nil
	}
	endpoint := acc + ".r2.cloudflarestorage.com"
	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(key, secret, ""),
		Secure: true,
		Region: "auto",
	})
	if err != nil {
		return nil, fmt.Errorf("r2 client: %w", err)
	}
	return &R2{client: cli, bucket: bucket, publicURL: pub}, nil
}

// Enabled — настроено ли хранилище.
func (r *R2) Enabled() bool { return r != nil && r.client != nil }

// Put загружает объект и возвращает публичный URL для отдачи.
func (r *R2) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) (string, error) {
	_, err := r.client.PutObject(ctx, r.bucket, key, body, size, minio.PutObjectOptions{
		ContentType:  contentType,
		CacheControl: "public, max-age=31536000, immutable",
	})
	if err != nil {
		return "", err
	}
	return r.publicURL + "/" + strings.TrimLeft(key, "/"), nil
}

// PublicHost — хост публичного URL (для CSP media-src/connect-src).
func (r *R2) PublicHost() string {
	if r == nil {
		return ""
	}
	return r.publicURL
}
