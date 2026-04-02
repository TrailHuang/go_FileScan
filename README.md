# Go 文件病毒扫描程序

一个高性能的Go语言文件病毒扫描程序，支持实时监控目录、动态学习表匹配和多种扫描模式。

## 功能特性

- ✅ **实时文件监控** - 监控指定目录的文件变化
- ✅ **动态学习表** - 支持热加载MD5病毒特征库
- ✅ **智能扫描策略** - 优先使用学习表匹配，未匹配则标记为白样本
- ✅ **高性能并发** - 支持多线程并发扫描
- ✅ **多种输出格式** - 支持JSON、文本、CSV格式输出
- ✅ **优雅关闭** - 支持信号处理和资源清理
- ✅ **跨平台支持** - 支持Linux、Windows、macOS
- ✅ **三种运行模式** - watch(实时监控)、scan(定时扫描)、once(单次扫描)

## 快速开始

### 1. 安装依赖

```bash
# 安装Go依赖
cd go_FileScan
go mod tidy

# 构建程序
go build -o filescan ./cmd/filescan
```

### 2. 配置程序

编辑 `config.yaml` 配置文件：

```yaml
# 文件病毒扫描程序配置
scanner:
  # 监控目录路径
  watch_directories:
    - /tmp/scan_directory
    - /var/www/uploads

  # 学习表文件路径
  learning_table_path: ./md5hash.txt

  # 扫描配置
  scan:
    max_concurrent_scans: 10
    file_size_limit: 100MB
    scan_timeout: 60s

# 输出配置
output:
  format: json
  file: ./scan_results.json
  include_clean_files: false
```

### 3. 准备学习表

学习表格式（每行一条记录）：
```
MD5哈希:文件大小:病毒名称
00000FCDD49303EA1AEB60EDC8499382:68:Gen:Variant.Ulise.85016
0000AD7FCCCD704C43C813D30932FAFC:63:Trojan.GenericKD.32729184
```

**注意**:
- 第一列为文件的MD5哈希值（大写）
- 第二列为文件大小
- 第三列为病毒名称
- 支持动态追加，程序会自动重新加载

### 4. 运行程序

#### 实时监控模式（默认）
```bash
./filescan
```
- 监控配置的目录
- 实时捕获文件创建、修改、权限变更事件
- 动态监控学习表文件变化

#### 单次扫描模式
```bash
./filescan --mode once --dir /path/to/scan
```
- 执行一次性目录扫描
- 扫描完成后立即退出
- 不监控学习表文件

#### 定时扫描模式
```bash
./filescan --mode scan
```
- 每10秒遍历一次监控目录
- 检测文件修改时间变化
- 只扫描新增或修改的文件

#### 命令行选项
```bash
./filescan --help

选项:
  -config string    配置文件路径 (默认 "config.yaml")
  -mode string      运行模式: watch, scan, once (默认 "watch")
  -dir string       扫描目录（覆盖配置文件）
  -output string    输出文件路径（覆盖配置文件）
  -format string    输出格式: json, text, csv（覆盖配置文件）
  -version          显示版本信息
```

## 程序架构

```
go_FileScan/
├── cmd/filescan/main.go          # 主程序入口
├── pkg/
│   ├── config/                   # 配置管理
│   ├── learning/                 # 学习表管理
│   ├── scanner/                  # 文件扫描引擎
│   ├── watcher/                  # 目录监控
│   └── output/                   # 结果输出
├── config.yaml                   # 配置文件
├── learning_table.txt            # 学习表示例
└── README.md                     # 说明文档
```

## 核心模块说明

### 1. 配置管理 (pkg/config)
- 支持YAML格式配置文件
- 动态配置加载
- 环境变量覆盖支持

### 2. 学习表管理 (pkg/learning)
- 动态加载和解析学习表
- 轮询方式监控文件变化（每2秒检查）
- 自动重载机制
- 线程安全的MD5查找

### 3. 文件扫描引擎 (pkg/scanner)
- MD5计算和匹配
- 并发扫描控制
- 超时和错误处理
- 文件大小限制

### 4. 目录监控 (pkg/watcher)
- 基于fsnotify的实时监控
- 递归目录监控
- 文件事件处理（CREATE, WRITE, CHMOD, REMOVE）

### 5. 结果输出 (pkg/output)
- 多种输出格式支持
- 文件输出和标准输出
- 扫描统计和摘要

## 运行模式详解

### Watch 模式（实时监控）
```bash
./filescan --mode watch
```

**特点**:
- 使用 fsnotify 实时监控文件系统事件
- 捕获文件创建、修改、权限变更
- 动态监控学习表文件变化
- 适合长期运行的监控场景

**适用场景**:
- 持续监控上传目录
- 实时监控关键文件夹
- 长期运行的守护进程

### Scan 模式（定时扫描）
```bash
./filescan --mode scan
```

**特点**:
- 每10秒遍历一次监控目录
- 基于文件修改时间检测变化
- 只扫描新增或修改的文件
- 自动记录文件最后修改时间

**适用场景**:
- 定期安全检查
- 资源受限环境
- 批量文件扫描

### Once 模式（单次扫描）
```bash
./filescan --mode once --dir /path/to/scan
```

**特点**:
- 执行一次性目录扫描
- 扫描完成后立即退出
- 不监控学习表文件
- 适合脚本调用

**适用场景**:
- CI/CD 集成
- 一次性安全检查
- 批量文件扫描

## 性能优化

### 并发控制
- 使用goroutine池控制并发扫描数量
- 避免文件句柄泄露
- 合理的超时设置

### 内存管理
- 流式MD5计算，避免大文件内存占用
- 学习表使用内存映射优化查找
- 结果通道缓冲控制

### 错误处理
- 完善的错误恢复机制
- 优雅的资源清理
- 详细的错误日志

## 部署建议

### 生产环境配置

1. **学习表更新**
   - 定期更新病毒特征库
   - 使用版本控制管理学习表
   - 测试新规则后再上线

2. **监控目录选择**
   - 选择高风险的目录进行监控
   - 避免监控系统关键目录
   - 考虑文件数量和大小限制

3. **资源限制**
   - 设置合理的文件大小限制
   - 控制并发扫描数量
   - 配置适当的超时时间

### 系统集成

```bash
# 创建系统服务
sudo cp filescan /usr/local/bin/
sudo cp config.yaml /etc/filescan/

# 创建systemd服务文件
sudo vim /etc/systemd/system/filescan.service
```

示例 systemd 服务文件：
```ini
[Unit]
Description=Go File Scanner Service
After=network.target

[Service]
Type=simple
User=nobody
ExecStart=/usr/local/bin/filescan -config /etc/filescan/config.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## 故障排除

### 常见问题

1. **学习表加载失败**
   - 检查文件路径和权限
   - 验证学习表格式是否正确
   - 确认文件编码为UTF-8

2. **文件监控不工作**
   - 检查目录是否存在和可访问
   - 确认文件系统支持inotify
   - 查看系统inotify限制

3. **并发扫描性能问题**
   - 调整 max_concurrent_scans 参数
   - 检查磁盘I/O性能
   - 考虑增加文件大小限制

### 调试模式

启用详细日志输出：
```bash
./filescan --config debug_config.yaml
```

查看学习表重载日志：
```
开始轮询监控学习表文件变化: ./md5hash.txt
初始文件状态 - 修改时间: 2024-01-01 12:00:00, 文件大小: 1024 bytes
检测到文件变化 - 修改时间: 2024-01-01 12:05:00, 文件大小: 2048 bytes
学习表重载成功! 记录数: 100 -> 101
```

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request来改进这个项目。
