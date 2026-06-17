BINARY := chainview
BIN_DIR := bin
PKG     := ./cmd/chainview

.PHONY: setup run build test lint vet clean

## setup: descarga dependencias y herramientas
setup:
	go mod download
	go mod tidy

## run: arranca la TUI sin compilar a disco
run:
	go run $(PKG)

## build: compila el binario en bin/
build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) $(PKG)

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
