.PHONY: generate test build web-build

generate:
	go generate ./ent

test: generate
	go test ./...

build: generate
	mkdir -p bin
	go build -trimpath -ldflags="-s -w" -o bin/xpanel-api ./cmd/api
	go build -trimpath -ldflags="-s -w" -o bin/xpanel-worker ./cmd/worker

web-build:
	cd web && pnpm install --frozen-lockfile && pnpm build

