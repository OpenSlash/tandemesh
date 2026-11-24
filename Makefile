GO ?= go
BIN := bin

.PHONY: build run agentd test vet web clean

build:
	mkdir -p $(BIN)
	$(GO) build -o $(BIN)/tandemesh-plane ./cmd/tandemesh-plane
	$(GO) build -o $(BIN)/tandemesh-agentd ./cmd/tandemesh-agentd
	$(GO) build -o $(BIN)/tandemesh ./cmd/tandemesh

run: build
	$(BIN)/tandemesh-plane

agentd: build
	$(BIN)/tandemesh-agentd -name dev-1

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

web:
	cd web && npm install && npm run build

clean:
	rm -rf $(BIN)
