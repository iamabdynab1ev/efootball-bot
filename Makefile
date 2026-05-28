.PHONY: build build-web run dev

build-web:
	cd web && npm run build

build: build-web
	go build -o bot ./cmd/bot

run: build
	./bot

dev:
	cd web && npm run dev
