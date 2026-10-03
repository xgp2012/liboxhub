# Boxli Hub 构建
#
# 目标：产出一个自包含的后端二进制，把前端产物（SSR bundle + 静态资源）内嵌进去。
#
# 为什么必须按顺序：Go 的 go:embed 在**编译期**读取 backend/internal/web/dist。
# 若前端产物尚未生成，embed 会**静默**嵌入空目录——编译不报错，
# 直到运行时才发现 404。因此这里把前端构建设为后端构建的前置依赖。

SHELL := /bin/bash

BACKEND_DIR  := backend
FRONTEND_DIR := frontend
DIST_DIR     := $(BACKEND_DIR)/internal/web/dist
BIN          := $(BACKEND_DIR)/boxli-hub

.PHONY: all build backend frontend fmt vet test clean dev-backend dev-frontend help

all: build

## build: 构建前端产物 + 内嵌它们的后端二进制（推荐入口）
build: backend

## backend: 只构建后端（会先确保前端产物存在）
backend: frontend
	@echo "▸ 编译后端（内嵌前端产物）…"
	cd $(BACKEND_DIR) && go build -o boxli-hub ./cmd/hub
	@echo "✓ 产物: $(BIN)"
	@ls -lh $(BIN) | awk '{print "  大小: " $$5}'

## frontend: 构建 Nuxt 产物并打包成单个 SSR bundle
frontend:
	@echo "▸ 构建前端（Nuxt build + esbuild 打包）…"
	# node_modules/.bin 下的包装脚本在部分环境（如本机）缺少可执行位，
	# 直接 `npm run build` 会得到 "nuxt: Permission denied"。
	# 这里显式补上，避免每次全新 clone 都要手工 chmod。
	@chmod +x $(FRONTEND_DIR)/node_modules/.bin/* 2>/dev/null || true
	cd $(FRONTEND_DIR) && npm run build
	cd $(FRONTEND_DIR) && npm run build:ssr-bundle

## fmt: 格式化 Go 代码
fmt:
	cd $(BACKEND_DIR) && go fmt ./...

## vet: 静态检查
vet:
	cd $(BACKEND_DIR) && go vet ./...

## test: 运行全部 Go 测试
test:
	cd $(BACKEND_DIR) && go test ./...

## clean: 清理构建产物
clean:
	rm -f $(BIN)
	rm -rf $(DIST_DIR) $(FRONTEND_DIR)/.output $(FRONTEND_DIR)/.nuxt
	@echo "已清理构建产物"

## dev-backend: 本地起后端（读取 backend/hub.toml）
dev-backend:
	cd $(BACKEND_DIR) && go run ./cmd/hub --config hub.toml

## dev-frontend: 本地起前端（端口 3011，勿用 3000——那是 Forgejo）
dev-frontend:
	cd $(FRONTEND_DIR) && PORT=3011 nuxt dev

## help: 显示可用目标
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
