# ⚽ eFootball League Bot

Telegram бот для управления футбольными лигами в eFootball 2026.  
Go + PostgreSQL + Goose миграции. Чистая архитектура, готов к расширению.

---

## 📁 Структура проекта

```
efootball-bot/
├── cmd/bot/
│   └── main.go                    # точка входа, роутинг команд
│
├── internal/
│   ├── bot/
│   │   └── handlers/
│   │       └── handlers.go        # /start /join /result /confirm
│   │
│   ├── models/
│   │   ├── user.go                # User
│   │   ├── league.go              # Season, League, LeagueMember
│   │   └── match.go               # Match, Dispute
│   │
│   ├── repository/
│   │   ├── interfaces.go          # интерфейсы (UserRepo, LeagueRepo, MatchRepo)
│   │   ├── user_postgres.go       # реализация для users
│   │   └── match_postgres.go      # реализация для matches
│   │
│   └── service/
│       ├── match.go               # логика результатов + начисление очков
│       └── schedule.go            # генератор расписания round-robin
│
├── migrations/
│   ├── 001_create_users.sql
│   ├── 002_create_seasons.sql
│   ├── 003_create_leagues.sql
│   ├── 004_create_league_members.sql
│   ├── 005_create_matches.sql
│   ├── 006_create_disputes.sql
│   └── 007_create_admins.sql
│
├── config/
│   └── config.go
│
├── .env.example
├── docker-compose.yml
└── Dockerfile
```

---

## 🗄️ Схема БД

```
users ──────────────────────────────── главная таблица игроков
  telegram_id  BIGINT UNIQUE           всегда есть у всех
  display_name VARCHAR(64)             сам ввёл при /start
  username     VARCHAR(64) nullable    @username если есть

seasons ────────────────────────────── сезоны (Сезон 1, Сезон 2...)
  status: pending | active | finished

leagues ────────────────────────────── лиги (АПЛ, Ла Лига...)
  level:       1=нац.лига, 2=ЛЧ, 3=ЛЕ
  rounds_type: single | double (один круг или дом+выезд)
  status: registration | active | finished

league_members ─────────────────────── участники лиги + статистика
  status: pending | approved | rejected
  points, wins, draws, losses, goals_for, goals_against, position

matches ────────────────────────────── матчи (сердце системы)
  status:
    scheduled        → матч назначен
    pending_confirm  → хозяин ввёл счёт, ждём гостя
    disputed         → гость не согласен, ждём хозяина снова
    confirmed        → оба согласились, очки начислены
    cancelled        → отменён админом

disputes ───────────────────────────── споры по результатам
  resolved_by: telegram_id админа который решил

admins ─────────────────────────────── список администраторов
```

---

## 🔄 Флоу результата матча

```
Матч сыгран
    │
    ▼
Хозяин → /result <matchID> 3:1
    │
    ▼
Бот → Гостю: "Хозяин сообщил 3:1. Подтверди?"
    │
    ├─── Гость /confirm_<matchID>
    │         │
    │         ▼
    │    Результат сохранён
    │    Очки начислены
    │    Отчёт в группу: "✅ Матч сыгран! ⚽ Алишер 3—1 Бобур"
    │
    └─── Гость /dispute_<matchID>
              │
              ▼
         Бот → Хозяину: "Соперник не согласен. Введи счёт заново"
              │
              ▼
         [цикл повторяется]
              │
              ▼  (если 3+ спора)
         Уведомление Админу → ты решаешь вручную
```

---

## 🤖 Команды бота

### Игроки
| Команда | Описание |
|---------|----------|
| `/start` | Регистрация, ввод имени |
| `/join АПЛ` | Заявка в лигу (ждёт одобрения) |
| `/result <id> 3:1` | Ввести счёт матча (только хозяин) |
| `/confirm_<id>` | Подтвердить результат (только гость) |
| `/dispute_<id>` | Не согласен с результатом |
| `/schedule` | Моё расписание матчей |
| `/table` | Турнирная таблица |
| `/stats` | Моя статистика |

### Админ (только ты)
| Команда | Описание |
|---------|----------|
| `/admin approve <userID> <leagueID>` | Одобрить заявку |
| `/admin reject <userID> <leagueID>` | Отклонить заявку |
| `/admin pending` | Список ожидающих одобрения |
| `/admin draw <leagueID>` | Жеребьёвка — сгенерировать расписание |
| `/admin resolve <matchID> 2:1` | Решить спор вручную |
| `/admin disputes` | Список активных споров |

---

## 🚀 Запуск

```bash
# 1. Скопировать .env
cp .env.example .env
# Заполнить BOT_TOKEN и POSS_DSN

# 2. Запустить через Docker
docker-compose up -d

# Или локально:
go run ./cmd/bot
```

---

## 📈 Алгоритм таблицы (как ФИФА)

Сортировка:
1. Очки (победа=3, ничья=1, поражение=0)
2. Разница голов (забитые − пропущенные)
3. Забитые голы
4. Личные встречи (если всё равно)

---

## 🗺️ Дорожная карта

- [x] БД схема + миграции
- [x] Регистрация игроков
- [x] Флоу результатов (claim → confirm/dispute)
- [x] Начисление очков
- [x] Генератор расписания (round-robin, дом+выезд)
- [ ] /table красивый вывод
- [ ] /schedule команда
- [ ] Admin handlers полностью
- [ ] Несколько лиг параллельно
- [ ] Лига Чемпионов (групповой этап)
- [ ] Повышение/вылет между лигами
- [ ] Плей-офф
