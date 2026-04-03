#!/bin/bash
# Go文件病毒扫描程序安装脚本

set -e

PROJECT_NAME="filescan"
VERSION="1.0.0"
INSTALL_DIR="/home/virus_scan"
CONFIG_DIR="/home/virus_scan"

# 检测系统架构
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l) ARCH="arm" ;;
    *) echo "不支持的架构: $ARCH"; exit 1 ;;
esac

# 颜色定义
RED="[0;31m"
GREEN="[0;32m"
YELLOW="[1;33m"
NC="[0m" # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 主安装流程
main() {
    log_info "开始安装Go文件病毒扫描程序 v$VERSION"

    # 检查是否已安装
    if [ -f "$INSTALL_DIR/filescan" ]; then
        log_warn "程序已安装，将进行覆盖安装"
    fi


    # 创建安装目录
    log_info "创建安装目录: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"

    # 复制程序文件
    log_info "复制程序文件..."
    if [ -f "./filescan-linux-$ARCH" ]; then
        cp "./filescan-linux-$ARCH" "$INSTALL_DIR/filescan"
    elif [ -f "./filescan" ]; then
        cp "./filescan" "$INSTALL_DIR/filescan"
    else
        log_error "未找到程序文件 filescan-linux-$ARCH 或 filescan"
        exit 1
    fi

    chmod +x "$INSTALL_DIR/filescan"

    # 创建配置目录
    log_info "创建配置目录: $CONFIG_DIR"
    mkdir -p "$CONFIG_DIR"

    # 复制配置文件
    if [ -f "./config.yaml" ]; then
        if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
            log_info "复制配置文件..."
            cp "./config.yaml" "$CONFIG_DIR/config.yaml"
        else
            log_warn "配置文件已存在，跳过复制"
        fi
    fi

    # 复制学习表
    if [ -f "./md5hash.txt" ]; then
        if [ ! -f "$CONFIG_DIR/md5hash.txt" ]; then
            log_info "复制学习表文件..."
            cp "./md5hash.txt" "$CONFIG_DIR/md5hash.txt"
        else
            log_warn "学习表文件已存在，跳过复制"
        fi
    fi

    # 创建隔离目录
    log_info "创建隔离目录: $CONFIG_DIR/quarantine"
    mkdir -p "$CONFIG_DIR/quarantine"

    log_info "安装完成!"
    echo ""
    echo "程序路径: $INSTALL_DIR/filescan"
    echo "配置文件: $CONFIG_DIR/config.yaml"
    echo "学习表: $CONFIG_DIR/learning_table.txt"
    echo "隔离目录: $CONFIG_DIR/quarantine"
    echo ""
    echo "使用方法:"
    echo "  $INSTALL_DIR/filescan --config $CONFIG_DIR/config.yaml"

    echo "或使用systemd服务:"
    echo "  sudo cp filescan.service /etc/systemd/system/"
    echo "  sudo systemctl daemon-reload"
    echo "  sudo systemctl start filescan"
    echo "  sudo systemctl enable filescan"
}

main "$@"
