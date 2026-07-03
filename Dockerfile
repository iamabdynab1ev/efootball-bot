FROM node:20-alpine AS web-builder
WORKDIR /app/web
# NEXT_PUBLIC_* переменные встраиваются в статический экспорт во время сборки —
# их нужно передать как build arg (рантайм-секреты тут уже не сработают).
ARG NEXT_PUBLIC_GOOGLE_CLIENT_ID
ENV NEXT_PUBLIC_GOOGLE_CLIENT_ID=$NEXT_PUBLIC_GOOGLE_CLIENT_ID
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# go.mod требует Go 1.25 (зависимости whatsmeow) — образ должен совпадать.
FROM golang:1.25-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/cmd/bot/ui ./cmd/bot/ui
RUN go build -o bot ./cmd/bot

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /app/bot ./bot
COPY migrations/ ./migrations/
EXPOSE 8080
CMD ["./bot"]
