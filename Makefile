.PHONY: build build-web run dev dev-all

build-web:
	cd web && npm run build

build: build-web
	go build -o bot ./cmd/bot

run: build
	./bot

dev:
	cd web && npm run dev

dev-all:
	@bash dev.sh
