BINARY := chainview
BIN_DIR := bin
PKG     := ./cmd/chainview

# Información de build inyectada en el binario (la lee `chainview --version`).
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

.PHONY: setup run build test lint vet clean

## setup: descarga dependencias y herramientas
setup:
	go mod download
	go mod tidy

## run: arranca la TUI sin compilar a disco
run:
	go run $(PKG)

## build: compila el binario en bin/ con la info de versión
build:
	mkdir -p $(BIN_DIR)
	go build -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY) $(PKG)

## test: ejecuta los tests con el detector de carreras
test:
	go test -race ./...

## vet: análisis estático del toolchain de Go
vet:
	go vet ./...

## lint: linter agregado (requiere golangci-lint instalado)
lint:
	golangci-lint run

## clean: elimina artefactos de build
clean:
	rm -rf $(BIN_DIR)
