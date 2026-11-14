GO ?= go
BIN := bin

.PHONY: build run agentd test vet web clean

build:
	mkdir -p $(BIN)
	$(GO) build -o $(BIN)/orca-plane ./cmd/orca-plane
	$(GO) build -o $(BIN)/orca-agentd ./cmd/orca-agentd
	$(GO) build -o $(BIN)/orca ./cmd/orca

run: build
	$(BIN)/orca-plane

agentd: build
	$(BIN)/orca-agentd -name dev-1

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

web:
	cd web && npm install && npm run build

clean:
	rm -rf $(BIN)
