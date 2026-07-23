APP        := ticking-clue
ENTRYPOINT := ./client/desktop
VERSION    := $(shell git describe --tags --always --dirty)
DEBUG      ?= false
LDFLAGS     = -ldflags "-X main.version=$(VERSION) -X main.debug=$(DEBUG)"
BIN_DIR    := bin
WASM_DIR   := client/wasm

.PHONY: help run build wasm serve clean proto convert-graph

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

run: DEBUG = true
run: ## Run desktop client with debug enabled
	go run $(LDFLAGS) $(ENTRYPOINT)

build: ## Build binary for current OS into bin/
	go build $(LDFLAGS) -o $(BIN_DIR)/$(APP) $(ENTRYPOINT)

wasm: ## Build wasm into client/wasm
	GOOS=js GOARCH=wasm go build $(LDFLAGS) -o $(WASM_DIR)/$(APP).wasm $(ENTRYPOINT)
	@GOROOT=$$(go env GOROOT); \
	if [ -f "$$GOROOT/lib/wasm/wasm_exec.js" ]; then \
		cp "$$GOROOT/lib/wasm/wasm_exec.js" $(WASM_DIR)/wasm_exec.js; \
	else \
		cp "$$GOROOT/misc/wasm/wasm_exec.js" $(WASM_DIR)/wasm_exec.js; \
	fi

serve: DEBUG = true
serve: wasm ## Build wasm with debug and serve client/wasm on :8080
	go run github.com/eliben/static-server@latest -port 8080 $(WASM_DIR)

proto: ## Generate Go code from proto files
	protoc --proto_path=./proto --go_out=proto/gen graph.proto

convert-graph: ## Convert neo4j json exports into binary protobuf graphs
	go run ./tools/graphconverter

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) $(WASM_DIR)/$(APP).wasm $(WASM_DIR)/wasm_exec.js
