# ============================================
# Makefile - API IAM Platform (EduGo)
# ============================================

# Variables
APP_NAME=api-iam-platform
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR=bin
COVERAGE_DIR=coverage
MAIN_PATH=./cmd/main.go

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

# Colors
YELLOW=\033[1;33m
GREEN=\033[1;32m
BLUE=\033[1;34m
RED=\033[1;31m
RESET=\033[0m

# Load .env file if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

.DEFAULT_GOAL := help

# ============================================
# Main Targets
# ============================================

help: ## Mostrar esta ayuda
	@echo "$(BLUE)$(APP_NAME) - Comandos disponibles:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'

build: ## Compilar binario
	@echo "$(YELLOW)Generando Swagger docs...$(RESET)"
	@swag init -g cmd/main.go -o docs --parseDependency --parseInternal 2>/dev/null || true
	@echo "$(YELLOW)Compilando $(APP_NAME)...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "$(GREEN)Binario: $(BUILD_DIR)/$(APP_NAME) ($(VERSION))$(RESET)"

build-debug: ## Compilar binario para debugging (sin optimizaciones)
	@echo "$(YELLOW)Compilando $(APP_NAME) para debug...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -gcflags "all=-N -l" $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "$(GREEN)Binario para debug: $(BUILD_DIR)/$(APP_NAME)$(RESET)"

run: ## Ejecutar en modo desarrollo
	@echo "$(YELLOW)Generando Swagger docs...$(RESET)"
	@swag init -g cmd/main.go -o docs --parseDependency --parseInternal 2>/dev/null || true
	@echo "$(YELLOW)Ejecutando $(APP_NAME)...$(RESET)"
	@$(GOCMD) run $(MAIN_PATH)

dev: deps run ## Desarrollo completo

# ============================================
# Testing
# ============================================

test-unit: ## Tests unitarios
	@echo "$(YELLOW)Ejecutando tests unitarios...$(RESET)"
	@$(GOTEST) -v -short -race ./internal/... -timeout 2m
	@echo "$(GREEN)Tests unitarios completados$(RESET)"

test-integration: ## Tests de integracion (con testcontainers)
	@echo "$(YELLOW)Ejecutando tests de integracion...$(RESET)"
	@RUN_INTEGRATION_TESTS=true $(GOTEST) -v -tags=integration ./test/integration/... -timeout 10m
	@echo "$(GREEN)Tests de integracion completados$(RESET)"

test-all: test-unit test-integration ## Ejecutar unitarios + integracion

coverage-report: ## Reporte de cobertura
	@echo "$(YELLOW)Generando reporte de cobertura...$(RESET)"
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./internal/... -timeout 5m || true
	@$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -20
	@echo "$(GREEN)Reporte: $(COVERAGE_DIR)/coverage.html$(RESET)"

coverage-check: ## Verificar cobertura minima (33%)
	@echo "$(YELLOW)Verificando cobertura minima...$(RESET)"
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./internal/... -timeout 5m
	@COVERAGE=$$($(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Cobertura: $${COVERAGE}%"; \
	if [ $$(echo "$${COVERAGE} < 33" | bc -l) -eq 1 ]; then \
		echo "$(RED)Cobertura por debajo del umbral de 33%$(RESET)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Cobertura cumple umbral minimo$(RESET)"

# ============================================
# Code Quality
# ============================================

fmt: ## Formatear codigo
	@echo "$(YELLOW)Formateando codigo...$(RESET)"
	@$(GOFMT) -w .
	@echo "$(GREEN)Codigo formateado$(RESET)"

vet: ## Analisis estatico
	@echo "$(YELLOW)Ejecutando go vet...$(RESET)"
	@$(GOVET) ./...
	@echo "$(GREEN)Analisis estatico completado$(RESET)"

lint: ## Linter completo
	@echo "$(YELLOW)Ejecutando golangci-lint...$(RESET)"
	@golangci-lint run --timeout=5m || echo "$(YELLOW)Instalar con: make tools$(RESET)"

audit: ## Auditoria de calidad completa
	@echo "$(BLUE)=== AUDITORIA ===$(RESET)"
	@echo "$(YELLOW)1. Verificando go.mod...$(RESET)"
	@$(GOMOD) verify
	@echo "$(YELLOW)2. Formato...$(RESET)"
	@test -z "$$($(GOFMT) -l .)" || (echo "$(RED)Sin formatear:$(RESET)" && $(GOFMT) -l .)
	@echo "$(YELLOW)3. Vet...$(RESET)"
	@$(GOVET) ./...
	@echo "$(YELLOW)4. Tests...$(RESET)"
	@$(GOTEST) -race -vet=off ./...
	@echo "$(GREEN)Auditoria completada$(RESET)"

# ============================================
# Dependencies
# ============================================

deps: ## Descargar dependencias
	@echo "$(YELLOW)Instalando dependencias...$(RESET)"
	@$(GOMOD) download
	@echo "$(GREEN)Dependencias listas$(RESET)"

tidy: ## Limpiar go.mod
	@echo "$(YELLOW)Limpiando go.mod...$(RESET)"
	@$(GOMOD) tidy
	@echo "$(GREEN)go.mod actualizado$(RESET)"

tools: ## Instalar herramientas
	@echo "$(YELLOW)Instalando herramientas...$(RESET)"
	@go install github.com/swaggo/swag/cmd/swag@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(GREEN)Herramientas instaladas$(RESET)"

# ============================================
# Swagger
# ============================================

swagger: ## Regenerar Swagger
	@echo "$(YELLOW)Regenerando Swagger...$(RESET)"
	@swag init -g cmd/main.go -o docs --parseInternal
	@echo "$(GREEN)Swagger: http://localhost:8080/swagger/index.html$(RESET)"

# ============================================
# Docker
# ============================================

docker-build: ## Build imagen Docker
	@echo "$(YELLOW)Building imagen...$(RESET)"
	@docker build -t edugo/$(APP_NAME):$(VERSION) .
	@echo "$(GREEN)Imagen: edugo/$(APP_NAME):$(VERSION)$(RESET)"

docker-up: ## Levantar servicios con compose
	@docker-compose up -d
	@echo "$(GREEN)API: http://localhost:8080$(RESET)"

docker-down: ## Detener servicios
	@docker-compose down

docker-logs: ## Ver logs de servicios
	@docker-compose logs -f

# ============================================
# CI/CD
# ============================================

ci: fmt vet test-unit coverage-check ## Pipeline CI completo
	@echo "$(GREEN)CI completado$(RESET)"

# ============================================
# Development
# ============================================

dev-init: deps ## Inicializar ambiente de desarrollo
	@echo "$(GREEN)Ambiente de desarrollo inicializado$(RESET)"

dev-status: ## Mostrar estado del proyecto
	@echo "$(BLUE)$(APP_NAME) - Estado$(RESET)"
	@echo "  Version:  $(VERSION)"
	@echo "  Go:       $$($(GOCMD) version | cut -d' ' -f3)"
	@echo "  Tests:    $$(find . -name '*_test.go' -type f | wc -l | xargs) archivos"

clean: ## Limpiar archivos generados
	@echo "$(YELLOW)Limpiando...$(RESET)"
	@rm -rf $(BUILD_DIR) $(COVERAGE_DIR)
	@echo "$(GREEN)Limpieza completada$(RESET)"

.PHONY: help build build-debug run dev test-unit test-integration test-all \
        coverage-report coverage-check fmt vet lint audit deps tidy tools \
        swagger docker-build docker-up docker-down docker-logs ci \
        dev-init dev-status clean
