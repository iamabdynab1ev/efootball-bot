package service

import (
	"context"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"time"
)

// NotificationService — внутри-приложенческие уведомления.
//
// В отличие от аудита, уведомления адресные и не должны теряться, поэтому
// запись синхронная (один multi-row INSERT на всех адресатов) — строка
// гарантированно есть в БД до ответа. Живая доставка через SSE — best-effort
// поверх персистентности: офлайн-получатель догрузит при следующем заходе.
type NotificationService struct {
	repo     repository.NotificationRepository
	publish  func(*models.Notification)
	keepDays int
}

const notifKeepDays = 60

// NewNotificationService. publish может быть nil (живая доставка не подключена).
func NewNotificationService(repo repository.NotificationRepository, publish func(*models.Notification)) *NotificationService {
	s := &NotificationService{repo: repo, publish: publish, keepDays: notifKeepDays}
	logger.Go("notif-prune", s.pruneLoop)
	return s
}

// Notify персистит уведомление каждому пользователю из userIDs и публикует его
// в личный SSE-топик. Дубликаты id и нули отбрасываются. Ошибку записи логируем,
// но наружу не пробрасываем — уведомление не должно валить основное действие.
func (s *NotificationService) Notify(ctx context.Context, userIDs []int64, typ, title, body, link string) {
	if s == nil || len(userIDs) == 0 {
		return
	}
	seen := make(map[int64]struct{}, len(userIDs))
	items := make([]*models.Notification, 0, len(userIDs))
	for _, uid := range userIDs {
		if uid == 0 {
			continue
		}
		if _, dup := seen[uid]; dup {
			continue
		}
		seen[uid] = struct{}{}
		items = append(items, &models.Notification{UserID: uid, Type: typ, Title: title, Body: body, Link: link})
	}
	if len(items) == 0 {
		return
	}
	if err := s.repo.CreateBatch(ctx, items); err != nil {
		logger.FromContext(ctx).Error("notification create failed", "error", err.Error(), "count", len(items))
		return
	}
	if s.publish != nil {
		for _, n := range items {
			s.publish(n)
		}
	}
}

// List отдаёт ленту уведомлений пользователя и счётчик непрочитанных.
func (s *NotificationService) List(ctx context.Context, userID, beforeID int64, limit int) ([]*models.Notification, int, error) {
	list, err := s.repo.ListByUser(ctx, userID, beforeID, limit)
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.repo.CountUnread(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return list, unread, nil
}

// MarkRead помечает прочитанными указанные id (или все при пустом списке).
func (s *NotificationService) MarkRead(ctx context.Context, userID int64, ids []int64) error {
	return s.repo.MarkRead(ctx, userID, ids)
}

func (s *NotificationService) pruneLoop() {
	time.Sleep(2 * time.Minute)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if n, err := s.repo.Prune(ctx, s.keepDays); err != nil {
			logger.FromContext(context.Background()).Error("notification prune failed", "error", err.Error())
		} else if n > 0 {
			logger.FromContext(context.Background()).Info("notifications pruned", "rows", n, "keep_days", s.keepDays)
		}
		cancel()
		time.Sleep(24 * time.Hour)
	}
}
