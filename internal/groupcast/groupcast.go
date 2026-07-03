// Package groupcast — общая шина групповых уведомлений турнира: одно событие
// (результат матча, жеребьёвка, напоминание, объявление) рассылается во все
// подключённые каналы — Telegram-группу и WhatsApp-группу.
package groupcast

import (
	"context"
	"log"
	"sync"
	"time"
)

// Sender — один канал доставки (Telegram-группа, WhatsApp-группа).
type Sender interface {
	Name() string
	// SendGroup отправляет обычный текст в подключённую группу канала.
	// Если группа не настроена — молча no-op (не ошибка).
	SendGroup(ctx context.Context, text string) error
}

// Hub — фан-аут по каналам. Отправка асинхронная: события не должны
// блокировать игровую логику (подтверждение счёта и т.п.).
type Hub struct {
	mu      sync.RWMutex
	senders []Sender
}

func NewHub() *Hub { return &Hub{} }

func (h *Hub) Add(s Sender) {
	if h == nil || s == nil {
		return
	}
	h.mu.Lock()
	h.senders = append(h.senders, s)
	h.mu.Unlock()
}

// Publish рассылает текст во все каналы. Безопасен на nil-хабе.
func (h *Hub) Publish(text string) {
	if h == nil || text == "" {
		return
	}
	h.mu.RLock()
	senders := make([]Sender, len(h.senders))
	copy(senders, h.senders)
	h.mu.RUnlock()
	for _, s := range senders {
		go func(s Sender) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			if err := s.SendGroup(ctx, text); err != nil {
				log.Printf("groupcast %s: %v", s.Name(), err)
			}
		}(s)
	}
}
