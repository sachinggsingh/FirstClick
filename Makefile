BIN_DIR=bin
BINARY=$(BIN_DIR)/firstclick

.PHONY: run test build clean

run:
	@echo "Starting FirstClick server on http://localhost:8080"
	@go run ./cmd/firstclick

test:
	@go test ./...

build:
	@mkdir -p $(BIN_DIR)
	@go build -o $(BINARY) ./cmd/firstclick

clean:
	@rm -rf $(BIN_DIR)