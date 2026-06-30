package service

import (
	"context"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"time"
)

// AuditService — асинхронная запись журнала действий.
//
// Запись не должна тормозить путь запроса, поэтому Log кладёт событие в буфер,
// а отдельная горутина пишет их пачками (один multi-row INSERT на пачку) —
// число обращений к БД остаётся постоянным при всплеске активности. Если буфер
// переполнен (экстремальная нагрузка), Log пишет синхронно: back-pressure
// вместо потери записи. После вставки каждая запись публикуется в живую ленту.
type AuditService struct {
	repo     repository.AuditRepository
	ch       chan *models.AuditEntry
	publish  func(*models.AuditEntry)
	keepDays int
}

const (
	auditBufferSize = 2048
	auditMaxBatch   = 100
	auditFlushEvery = 300 * time.Millisecond
	auditKeepDays   = 90
)

// NewAuditService запускает фоновых писателя и уборщика. publish может быть nil
// (живая лента не подключена) — тогда события только пишутся в БД.
func NewAuditService(repo repository.AuditRepository, publish func(*models.AuditEntry)) *AuditService {
	a := &AuditService{
		repo:     repo,
		ch:       make(chan *models.AuditEntry, auditBufferSize),
		publish:  publish,
		keepDays: auditKeepDays,
	}
	logger.Go("audit-writer", a.writeLoop)
	logger.Go("audit-prune", a.pruneLoop)
	return a
}

// List отдаёт ленту журнала по фильтру (для админ-выборки/истории).
func (a *AuditService) List(ctx context.Context, f models.AuditFilter) ([]*models.AuditEntry, error) {
	return a.repo.List(ctx, f)
}

// Log регистрирует действие. Никогда не блокирует горячий путь: при полном
// буфере падает в синхронную вставку, чтобы не потерять событие.
func (a *AuditService) Log(e *models.AuditEntry) {
	if e == nil || a == nil {
		return
	}
	select {
	case a.ch <- e:
	default:
		a.flush([]*models.AuditEntry{e})
	}
}

func (a *AuditService) writeLoop() {
	batch := make([]*models.AuditEntry, 0, auditMaxBatch)
	timer := time.NewTimer(auditFlushEvery)
	timer.Stop()
	for {
		select {
		case e := <-a.ch:
			batch = append(batch, e)
			// Добираем всё, что уже в буфере, без блокировки.
			for len(batch) < auditMaxBatch {
				select {
				case e2 := <-a.ch:
					batch = append(batch, e2)
				default:
					goto drained
				}
			}
		drained:
			if len(batch) >= auditMaxBatch {
				a.flush(batch)
				batch = batch[:0]
			} else {
				timer.Reset(auditFlushEvery)
			}
		case <-timer.C:
			if len(batch) > 0 {
				a.flush(batch)
				batch = batch[:0]
			}
		}
	}
}

func (a *AuditService) flush(batch []*models.AuditEntry) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.repo.InsertBatch(ctx, batch); err != nil {
		logger.FromContext(context.Background()).Error("audit insert failed", "error", err.Error(), "count", len(batch))
		return
	}
	if a.publish != nil {
		for _, e := range batch {
			a.publish(e)
		}
	}
}

func (a *AuditService) pruneLoop() {
	// Первая уборка через минуту после старта, далее раз в сутки — держим
	// таблицу в пределах лимита Neon без ручного вмешательства.
	time.Sleep(time.Minute)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if n, err := a.repo.Prune(ctx, a.keepDays); err != nil {
			logger.FromContext(context.Background()).Error("audit prune failed", "error", err.Error())
		} else if n > 0 {
			logger.FromContext(context.Background()).Info("audit pruned", "rows", n, "keep_days", a.keepDays)
		}
		cancel()
		time.Sleep(24 * time.Hour)
	}
}
