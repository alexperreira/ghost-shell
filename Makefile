BIN := bin/ghost-shell
NODE_VERSION ?= v22.22.0
export PATH := $(HOME)/.nvm/versions/node/$(NODE_VERSION)/bin:$(PATH)

.PHONY: build clean run

build:
	cd ui && pnpm install --frozen-lockfile && pnpm build
	mkdir -p bin
	go build -o $(BIN) ./cmd/ghost-shell

run: build
	./$(BIN)

clean:
	rm -rf bin/ internal/web/static/
