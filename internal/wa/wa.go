// Package wa — канал «WhatsApp-группа» через whatsmeow (протокол WhatsApp Web).
//
// ВАЖНО: это неофициальный клиент — используйте ОТДЕЛЬНЫЙ номер (не личный),
// Meta может заблокировать аккаунт за автоматизацию. Включается переменной
// окружения WA_ENABLED=1. Сессия хранится в Postgres (таблицы whatsmeow_*),
// вход — сканированием QR-кода на странице администратора.
package wa

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"efootball-bot/internal/repository"

	_ "github.com/lib/pq" // database/sql драйвер для sqlstore whatsmeow
	qrcode "github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Ключ настройки с JID подключённой WhatsApp-группы.
const settingGroupJID = "wa_group_jid"

// GroupInfo — группа, в которой состоит подключённый аккаунт.
type GroupInfo struct {
	JID  string `json:"jid"`
	Name string `json:"name"`
}

// Client — обёртка whatsmeow: статус подключения, QR для входа, выбор группы
// и отправка текста в неё (реализует groupcast.Sender).
type Client struct {
	cli      *whatsmeow.Client
	settings repository.SettingsRepository

	mu     sync.RWMutex
	qrPNG  []byte // текущий QR-код для входа (PNG), пуст когда не нужен
	status string // connecting | qr | connected | logged_out | error
	group  string // JID подключённой группы ("" — не выбрана)
}

// NewFromEnv создаёт клиент, если WA_ENABLED=1; иначе (nil, nil) — канал выключен.
func NewFromEnv(dsn string, settings repository.SettingsRepository) (*Client, error) {
	if os.Getenv("WA_ENABLED") != "1" {
		return nil, nil
	}
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "postgres", dsn, waLog.Noop)
	if err != nil {
		return nil, err
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, err
	}
	c := &Client{
		cli:      whatsmeow.NewClient(device, waLog.Noop),
		settings: settings,
		status:   "connecting",
	}
	if vals, err := settings.GetMany(ctx, []string{settingGroupJID}); err == nil {
		c.group = vals[settingGroupJID]
	}
	c.cli.AddEventHandler(func(evt any) {
		switch evt.(type) {
		case *events.Connected:
			c.setState("connected", nil)
		case *events.LoggedOut:
			// Аккаунт разлогинен (например, «выйти со всех устройств») —
			// нужен повторный вход по QR после рестарта.
			c.setState("logged_out", nil)
		}
	})
	go c.connectLoop()
	return c, nil
}

func (c *Client) setState(status string, qr []byte) {
	c.mu.Lock()
	c.status = status
	c.qrPNG = qr
	c.mu.Unlock()
}

// connectLoop подключается; пока аккаунт не залогинен — крутит QR-коды
// (каждый живёт ~минуту), после таймаута пробует заново.
func (c *Client) connectLoop() {
	for {
		if c.cli.Store.ID != nil {
			// Сессия уже есть — обычное подключение.
			if err := c.cli.Connect(); err != nil {
				log.Printf("wa: connect: %v", err)
				c.setState("error", nil)
				time.Sleep(30 * time.Second)
				continue
			}
			return
		}
		// Логина нет — получаем QR-коды и ждём сканирования.
		qrChan, err := c.cli.GetQRChannel(context.Background())
		if err != nil {
			log.Printf("wa: qr channel: %v", err)
			c.setState("error", nil)
			time.Sleep(30 * time.Second)
			continue
		}
		if err := c.cli.Connect(); err != nil {
			log.Printf("wa: connect(qr): %v", err)
			c.setState("error", nil)
			time.Sleep(30 * time.Second)
			continue
		}
		for evt := range qrChan {
			switch evt.Event {
			case "code":
				png, err := qrcode.Encode(evt.Code, qrcode.Medium, 512)
				if err != nil {
					log.Printf("wa: qr encode: %v", err)
					continue
				}
				c.setState("qr", png)
			case "success":
				c.setState("connected", nil)
				return
			}
		}
		// Канал закрылся без успеха (таймаут) — новая попытка через паузу.
		c.cli.Disconnect()
		c.setState("connecting", nil)
		time.Sleep(10 * time.Second)
	}
}

// Name — имя канала для groupcast.
func (c *Client) Name() string { return "whatsapp" }

// Status возвращает состояние подключения и JID выбранной группы.
func (c *Client) Status() (status, groupJID string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status, c.group
}

// QRPNG — текущий QR для входа (nil, если вход не требуется).
func (c *Client) QRPNG() []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.qrPNG
}

// Groups — группы, где состоит аккаунт (для выбора в админке).
func (c *Client) Groups(ctx context.Context) ([]GroupInfo, error) {
	if c.cli.Store.ID == nil {
		return nil, errors.New("аккаунт WhatsApp не подключён")
	}
	gs, err := c.cli.GetJoinedGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]GroupInfo, 0, len(gs))
	for _, g := range gs {
		out = append(out, GroupInfo{JID: g.JID.String(), Name: g.GroupName.Name})
	}
	return out, nil
}

// SetGroup выбирает группу для уведомлений (jid="" — отключить) и сохраняет выбор.
func (c *Client) SetGroup(ctx context.Context, jid string) error {
	if jid != "" {
		if _, err := types.ParseJID(jid); err != nil {
			return err
		}
	}
	c.mu.Lock()
	c.group = jid
	c.mu.Unlock()
	return c.settings.Set(ctx, settingGroupJID, jid)
}

// Logout отвязывает аккаунт WhatsApp (аналог «выйти» в связанных устройствах):
// сессия удаляется из базы, выбор группы сбрасывается, и снова поднимается
// QR-цикл — можно сразу привязать другой номер без рестарта сервиса.
func (c *Client) Logout(ctx context.Context) error {
	if c.cli.Store.ID == nil {
		return nil // и так не привязан
	}
	if err := c.cli.Logout(ctx); err != nil {
		return err
	}
	_ = c.SetGroup(ctx, "")
	c.setState("connecting", nil)
	go c.connectLoop()
	return nil
}

// SendGroup отправляет текст в выбранную группу (no-op, если не настроено).
func (c *Client) SendGroup(ctx context.Context, text string) error {
	c.mu.RLock()
	group, status := c.group, c.status
	c.mu.RUnlock()
	if group == "" || status != "connected" {
		return nil
	}
	jid, err := types.ParseJID(group)
	if err != nil {
		return err
	}
	_, err = c.cli.SendMessage(ctx, jid, &waProto.Message{Conversation: proto.String(text)})
	return err
}
