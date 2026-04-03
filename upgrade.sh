#!/bin/bash
# Go文件病毒扫描程序升级脚本

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

# 主升级流程
main() {
    log_info "开始升级Go文件病毒扫描程序到 v$VERSION"
    
    # 检查是否已安装
    if [ ! -f "$INSTALL_DIR/filescan" ]; then
        log_error "未找到已安装的程序文件: $INSTALL_DIR/filescan"
        log_info "请先运行 install.sh 进行安装"
        exit 1
    fi
    
    # 备份当前版本
    BACKUP_DIR="$CONFIG_DIR/backup_$(date +%Y%m%d_%H%M%S)"
    log_info "备份当前版本到: $BACKUP_DIR"
    mkdir -p "$BACKUP_DIR"
    cp "$INSTALL_DIR/filescan" "$BACKUP_DIR/filescan"
    
    # 复制新版本
    log_info "复制新版本程序..."
    if [ -f "./filescan-linux-$ARCH" ]; then
        cp "./filescan-linux-$ARCH" "$INSTALL_DIR/filescan"
    elif [ -f "./filescan" ]; then
        cp "./filescan" "$INSTALL_DIR/filescan"
    else
        log_error "未找到新版本程序文件 filescan-linux-$ARCH 或 filescan"
        exit 1
    fi
    
    chmod +x "$INSTALL_DIR/filescan"
    
    log_info "升级完成!"
    echo ""
    echo "程序路径: $INSTALL_DIR/filescan"
    echo "备份目录: $BACKUP_DIR"
    echo ""
    echo "如需回滚到旧版本:"
    echo "  cp $BACKUP_DIR/filescan $INSTALL_DIR/filescan"
}

main "$@"
