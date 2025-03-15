# Makefile for Performance-Raft-KV Project

# 项目配置
PROJECT_NAME := hraftd
VERSION := $(shell git describe --tags --always --dirty)
BUILD_DIR := build
DIST_DIR := dist

# Go 配置
GO := go
GO_FLAGS := -ldflags "-X main.Version=$(VERSION)"
GO_PACKAGES := ./...

# 平台配置
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# 依赖管理
DEPS_CMD := go mod

# 编译工具
COMPILER := $(GO)
COMPILER_FLAGS := $(GO_FLAGS)

# 测试配置
TEST_FLAGS := -v -race
COVERAGE_DIR := $(BUILD_DIR)/coverage

# 代码质量工具
LINT_CMD := golangci-lint

# 颜色定义
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m

# 默认目标
.PHONY: default
default: help

# 默认目标
.PHONY: all
all: clean deps build test lint

# 初始化项目依赖
.PHONY: deps
deps:
	@echo "$(GREEN)正在下载依赖...$(NC)"
	$(DEPS_CMD) download
	$(DEPS_CMD) tidy

# 清理编译产物
.PHONY: clean
clean:
	@echo "$(GREEN)清理编译产物...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)

# 多平台编译
#.PHONY: build
#build: clean deps
#	@echo "$(GREEN)开始多平台编译...$(NC)"
#	@mkdir -p $(BUILD_DIR)
#	@mkdir -p $(DIST_DIR)
#	$(foreach platform,$(PLATFORMS), \
#		$(eval OS=$(word 1,$(subst /, ,$(platform)))) \
#		$(eval ARCH=$(word 2,$(subst /, ,$(platform)))) \
#		$(eval OUTPUT=$(BUILD_DIR)/$(PROJECT_NAME)-$(OS)-$(ARCH)$(if $(filter windows,$(OS)),,.exe)) \
#		GOOS=$(OS) GOARCH=$(ARCH) $(COMPILER) build $(GO_FLAGS) -o $(OUTPUT) && \
#		tar -czvf $(DIST_DIR)/$(PROJECT_NAME)-$(OS)-$(ARCH).tar.gz $(OUTPUT); \
#	)
#	@echo "$(GREEN)编译完成，产物位于 $(DIST_DIR)$(NC)"

# 本地开发编译
.PHONY: dev
dev:
	@echo "$(GREEN)本地开发编译...$(NC)"
	$(COMPILER) build $(GO_FLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME)

# 运行单元测试
.PHONY: test
test:
	@echo "$(GREEN)运行单元测试...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test $(TEST_FLAGS) $(GO_PACKAGES) -coverprofile=$(COVERAGE_DIR)/coverage.out
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html

# 代码质量检查
.PHONY: lint
lint:
	@echo "$(GREEN)代码质量检查...$(NC)"
	$(LINT_CMD) run ./...

# 性能测试
.PHONY: bench
bench:
	@echo "$(GREEN)运行性能测试...$(NC)"
	$(GO) test -bench=. -benchmem $(GO_PACKAGES)

# 生成文档
.PHONY: docs
docs:
	@echo "$(GREEN)生成项目文档...$(NC)"
	godoc -http=:6060

# Docker 镜像构建
.PHONY: docker
docker:
	@echo "$(GREEN)构建 Docker 镜像...$(NC)"
	docker build -t $(PROJECT_NAME):$(VERSION) .

# 帮助信息
.PHONY: help
help:
	@echo "$(GREEN)可用的 Make 目标:$(NC)"
	@echo "  all       - 执行 clean, deps, build, test, lint"
	@echo "  deps      - 下载并整理依赖"
	@echo "  clean     - 清理编译产物"
	@echo "  build     - 多平台编译"
	@echo "  dev       - 本地开发编译"
	@echo "  test      - 运行单元测试"
	@echo "  lint      - 代码质量检查"
	@echo "  bench     - 运行性能测试"
	@echo "  docs      - 生成项目文档"
	@echo "  docker    - 构建 Docker 镜像"
	@echo "  help      - 显示帮助信息"

