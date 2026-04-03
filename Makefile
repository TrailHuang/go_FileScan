# Go文件病毒扫描程序 Makefile

# 项目信息
PROJECT_NAME := filescan
VERSION := 1.0.0
BUILD_TIME := $(shell date +%Y%m%d%H%M%S)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go配置
GO := go
GOFLAGS := -ldflags="-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

# 构建目录
BUILD_DIR := build
BIN_DIR := $(BUILD_DIR)/bin
DIST_DIR := $(BUILD_DIR)/dist

# 目标平台架构（只编译Linux）
ARCHS := amd64 arm64
OSES := linux

# 默认目标
.DEFAULT_GOAL := help

.PHONY: help build clean test install uninstall release install-script upgrade-script

# 显示帮助信息
help:
	@echo "Go文件病毒扫描程序构建系统"
	@echo ""
	@echo "可用目标:"
	@echo "  build          构建当前平台的可执行文件"
	@echo "  build-all      构建Linux平台的ARM和x86版本"
	@echo "  clean          清理构建文件"
	@echo "  test           运行测试"
	@echo "  install        安装到系统"
	@echo "  uninstall      从系统卸载"
	@echo "  release        创建Linux ARM/x86发布包"
	@echo "  install-script 生成安装脚本"
	@echo "  upgrade-script 生成升级脚本"
	@echo ""
	@echo "架构支持: $(ARCHS)"
	@echo "系统支持: $(OSES)"

# 创建构建目录
$(BUILD_DIR) $(BIN_DIR) $(DIST_DIR):
	@mkdir -p $@

# 构建当前平台
build: $(BIN_DIR)
	@echo "构建当前平台的可执行文件..."
	$(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(PROJECT_NAME) ./cmd/filescan
	@echo "构建完成: $(BIN_DIR)/$(PROJECT_NAME)"

# 构建Linux平台的ARM和x86版本
build-all: $(BIN_DIR)
	@echo "构建Linux平台的ARM和x86版本..."
	@for arch in $(ARCHS); do \
		echo "构建 linux/$$arch..."; \
		GOOS=linux GOARCH=$$arch $(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(PROJECT_NAME)-linux-$$arch ./cmd/filescan; \
	done
	@echo "Linux平台构建完成"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@echo "清理完成"

# 运行测试
test:
	@echo "运行测试..."
	$(GO) test -v ./...
	@echo "测试完成"

# 安装到系统
install: build
	@echo "安装到系统..."
	@if [ ! -f $(BIN_DIR)/$(PROJECT_NAME) ]; then \
		echo "错误: 请先运行 'make build' 构建程序"; \
		exit 1; \
	fi
	@sudo install -m 755 $(BIN_DIR)/$(PROJECT_NAME) /usr/local/bin/$(PROJECT_NAME)
	@sudo mkdir -p /etc/$(PROJECT_NAME)
	@if [ ! -f /etc/$(PROJECT_NAME)/config.yaml ]; then \
		sudo install -m 644 config.yaml /etc/$(PROJECT_NAME)/; \
	fi
	@echo "安装完成"
	@echo "程序路径: /usr/local/bin/$(PROJECT_NAME)"
	@echo "配置文件: /etc/$(PROJECT_NAME)/config.yaml"

# 从系统卸载
uninstall:
	@echo "从系统卸载..."
	@sudo rm -f /usr/local/bin/$(PROJECT_NAME)
	@echo "卸载完成"

# 创建Linux ARM/x86发布包（一起打包）

release: build-all $(DIST_DIR)
	@echo "创建Linux ARM和x86发布包..."
	@mkdir -p $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch

	# 复制ARM和x86版本
	@for arch in $(ARCHS); do \
		echo "复制 linux/$$arch 版本..."; \
		cp $(BIN_DIR)/$(PROJECT_NAME)-linux-$$arch $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/$(PROJECT_NAME)-linux-$$arch; \
	done

	# 复制配置文件和文档
	cp config.yaml $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/
	cp README.md $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/
	cp install.sh $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/
	cp upgrade.sh $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/
	cp md5hash.txt $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/

	# 创建安装说明
	@echo "# Go文件病毒扫描程序安装说明" > $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "## 版本信息" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "- 程序版本: $(VERSION)" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "- 构建时间: $(BUILD_TIME)" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "- 包含架构: amd64 (x86_64), arm64" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "## 安装步骤" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "### 1. 使用安装脚本（推荐）" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo '```bash' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo 'chmod +x install.sh' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo 'sudo ./install.sh' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo '```' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "### 2. 手动安装" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "#### 2.1 选择适合您系统的版本" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "- **x86_64系统**: 使用 filescan-linux-amd64" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "- **ARM64系统**: 使用 filescan-linux-arm64" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "#### 2.2 安装程序" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo '```bash' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo '# 复制到系统目录' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo 'sudo cp filescan-linux-$(shell uname -m) /usr/local/bin/filescan' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo 'sudo chmod +x /usr/local/bin/filescan' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo '' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo '# 创建配置目录' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo 'sudo mkdir -p /etc/filescan' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo 'sudo cp config.yaml /etc/filescan/' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo 'sudo cp learning_table.txt /etc/filescan/' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo '```' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "### 3. 启动服务" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo '```bash' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo '# 直接运行' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo 'filescan --config /etc/filescan/config.yaml' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo '' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo '# 或使用systemd服务' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo 'sudo systemctl start filescan' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo 'sudo systemctl enable filescan' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo '```' >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "## 升级说明" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md
	@echo "使用升级脚本进行升级：`sudo ./upgrade.sh`" >> $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch/INSTALL.md

	# 打包
	cd $(DIST_DIR) && tar -czf $(PROJECT_NAME)-$(VERSION)-linux-multiarch.tar.gz $(PROJECT_NAME)-$(VERSION)-linux-multiarch
	rm -rf $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-multiarch
	@echo "发布包创建完成: $(PROJECT_NAME)-$(VERSION)-linux-multiarch.tar.gz"


