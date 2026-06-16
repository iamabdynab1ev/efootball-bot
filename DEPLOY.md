# 🚀 Бесплатный деплой (Koyeb + Neon)

Цель: запустить приложение публично и **бесплатно**, чтобы было доступно всем,
с постоянно работающим Telegram-ботом (always-on) и HTTPS.

Стек: **Neon** (бесплатный PostgreSQL) + **Koyeb** (бесплатный хостинг, собирает
из `Dockerfile`). Бот и веб — один процесс (Go-бинарь раздаёт сайт и API).

---

## 1. База данных — Neon (бесплатно)

1. Зарегистрируйтесь на <https://neon.tech> → **Create project**.
2. Скопируйте **connection string** (Dashboard → Connection Details).
3. В конец добавьте `?sslmode=require`, например:
   ```
   postgres://user:pass@ep-xxx.eu-central-1.aws.neon.tech/efootball?sslmode=require
   ```
   Это будет переменная `POSTGRES_DSN`.

## 2. Залить код на GitHub

Из этой папки (вы уже залогинены в GitHub в своём терминале):
```bash
git push origin HEAD:main
```
Это отправит последнюю версию в ветку `main` репозитория
`github.com/iamabdynab1ev/efootball-bot`.

## 3. Хостинг — Koyeb (бесплатно, always-on)

1. Зарегистрируйтесь на <https://www.koyeb.com> (можно через GitHub).
2. **Create App** → источник **GitHub** → выбрать репозиторий `efootball-bot`,
   ветку `main`. Koyeb сам найдёт `Dockerfile`.
3. **Builder**: Dockerfile. **Exposed port**: `8080`.
4. **Build → Build arguments** (важно — Google Client ID встраивается в сайт
   на этапе сборки):
   | Build arg | Значение |
   |---|---|
   | `NEXT_PUBLIC_GOOGLE_CLIENT_ID` | ваш Google Client ID |
5. **Environment variables**:
   | Переменная | Значение |
   |---|---|
   | `APP_ENV` | `production` |
   | `POSTGRES_DSN` | строка из Neon (шаг 1) |
   | `JWT_SECRET` | любая случайная строка **≥ 32 символов** |
   | `BOT_TOKEN` | токен от @BotFather |
   | `GOOGLE_CLIENT_ID` | тот же Google Client ID |
   | `FRONTEND_URL` | `https://<имя-приложения>.koyeb.app` (узнаете после деплоя) |
   | `ADMIN_USERNAME` | логин супер-админа |
   | `ADMIN_PASSWORD` | пароль супер-админа |
6. **Deploy**. Koyeb соберёт образ и запустит. Миграции применятся сами,
   супер-админ создастся автоматически.

## 4. После первого деплоя

1. Koyeb выдаст адрес `https://<имя>.koyeb.app`.
2. Впишите его в переменную `FRONTEND_URL` → **Redeploy**.
3. **Google вход**: в [Google Cloud Console](https://console.cloud.google.com)
   → APIs & Services → Credentials → ваш OAuth Client →
   **Authorized JavaScript origins** добавьте `https://<имя>.koyeb.app`.

Готово — приложение доступно всем по адресу `https://<имя>.koyeb.app`.

---

## Заметки

- **Telegram-бот** работает сразу (long-poll, публичный URL ему не нужен).
- **Google-вход** заработает только после шага 4.3 (добавления origin).
- Бесплатный Neon может «засыпать» при простое — первый запрос после паузы
  чуть медленнее, это нормально.
- Обновление: `git push origin HEAD:main` → Koyeb пересоберёт автоматически.

## Альтернатива: свой сервер (Docker)

Если есть VPS с публичным IP/доменом:
```bash
cp .env.example .env   # заполнить переменные (POSTGRES_DSN=postgres://efootball:secret@postgres:5432/efootball?sslmode=disable)
docker compose up -d --build
```
> В `docker-compose.yml` у сервиса `bot` добавьте проброс порта
> `ports: ["8080:8080"]` и поставьте перед ним reverse-proxy (Caddy/nginx)
> для HTTPS.
