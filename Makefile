BIN_DIR = bin
SERVER_BIN = $(BIN_DIR)/druid-insight
SERVICE_BIN = $(BIN_DIR)/service
USERCTL_BIN = $(BIN_DIR)/userctl
DATASOURCE_SYNC_BIN = $(BIN_DIR)/datasource-sync

SERVER_MAIN = cmd/druid-insight/main.go
SERVICE_MAIN = cmd/service/main.go
USERCTL_MAIN = cmd/userctl/main.go
DATASOURCE_SYNC_MAIN = cmd/datasource-sync/main.go

# Toutes les sources .go du projet sauf dans bin/
GO_SOURCES := $(shell find . -type f -name '*.go' ! -path './bin/*')

.PHONY: all build run start stop reload clean test

all: build

build: $(SERVER_BIN) $(SERVICE_BIN) $(USERCTL_BIN) $(DATASOURCE_SYNC_BIN)

$(SERVER_BIN): $(GO_SOURCES)
	mkdir -p $(BIN_DIR)
	go build -o $(SERVER_BIN) $(SERVER_MAIN)

$(SERVICE_BIN): $(GO_SOURCES)
	mkdir -p $(BIN_DIR)
	go build -o $(SERVICE_BIN) $(SERVICE_MAIN)

$(USERCTL_BIN): $(GO_SOURCES)
	mkdir -p $(BIN_DIR)
	go build -o $(USERCTL_BIN) $(USERCTL_MAIN)

$(DATASOURCE_SYNC_BIN): $(GO_SOURCES)
	mkdir -p $(BIN_DIR)
	go build -o $(DATASOURCE_SYNC_BIN) $(DATASOURCE_SYNC_MAIN)

run: build
	$(SERVER_BIN)

start: build
	$(SERVICE_BIN) start

stop:
	$(SERVICE_BIN) stop

reload:
	$(SERVICE_BIN) reload

test:
	go test ./...

clean:
	rm -rf $(BIN_DIR)
