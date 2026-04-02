package learning

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type VirusRecord struct {
	MD5       string
	Size      string
	VirusName string
}

type LearningTable struct {
	filePath string
	records  map[string]*VirusRecord
	mu       sync.RWMutex
}

func NewLearningTable(filePath string, mode string) (*LearningTable, error) {
	lt := &LearningTable{
		filePath: filePath,
		records:  make(map[string]*VirusRecord),
	}

	if err := lt.load(); err != nil {
		return nil, err
	}

	if mode != "once" {
		if err := lt.startWatching(); err != nil {
			return nil, err
		}
	}

	return lt, nil
}

func (lt *LearningTable) load() error {
	file, err := os.Open(lt.filePath)
	if err != nil {
		return fmt.Errorf("failed to open learning table file: %w", err)
	}
	defer file.Close()

	lt.mu.Lock()
	defer lt.mu.Unlock()

	lt.records = make(map[string]*VirusRecord)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			return fmt.Errorf("invalid format at line %d: %s", lineNum, line)
		}

		record := &VirusRecord{
			MD5:       parts[0],
			Size:      parts[1],
			VirusName: parts[2],
		}

		lt.records[record.MD5] = record
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading learning table: %w", err)
	}

	return nil
}

func (lt *LearningTable) Lookup(md5 string) (*VirusRecord, bool) {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	record, exists := lt.records[md5]
	return record, exists
}

func (lt *LearningTable) startWatching() error {
	// 检查文件是否存在
	if _, err := os.Stat(lt.filePath); err != nil {
		return fmt.Errorf("learning table file does not exist: %w", err)
	}

	fmt.Printf("开始轮询监控学习表文件: %s\n", lt.filePath)

	// 使用轮询方式替代文件监控
	go lt.pollChanges()

	return nil
}

func (lt *LearningTable) pollChanges() {
	fmt.Printf("开始轮询监控学习表文件变化: %s\n", lt.filePath)

	// 记录上次修改时间和文件大小
	var lastModTime time.Time
	var lastFileSize int64

	// 初始化上次状态
	if fileInfo, err := os.Stat(lt.filePath); err == nil {
		lastModTime = fileInfo.ModTime()
		lastFileSize = fileInfo.Size()
		fmt.Printf("初始文件状态 - 修改时间: %v, 文件大小: %d bytes\n", lastModTime, lastFileSize)
	}

	ticker := time.NewTicker(2 * time.Second) // 每2秒检查一次
	defer ticker.Stop()

	for range ticker.C {
		// 检查文件是否存在
		fileInfo, err := os.Stat(lt.filePath)
		if err != nil {
			fmt.Printf("学习表文件不存在: %v\n", err)
			continue
		}

		currentModTime := fileInfo.ModTime()
		currentFileSize := fileInfo.Size()

		// 检查文件是否发生了变化
		if currentModTime.After(lastModTime) || currentFileSize != lastFileSize {
			fmt.Printf("检测到文件变化 - 修改时间: %v (上次: %v), 文件大小: %d bytes (上次: %d bytes)\n",
				currentModTime, lastModTime, currentFileSize, lastFileSize)

			// 更新状态
			lastModTime = currentModTime
			lastFileSize = currentFileSize

			// 记录重载前的记录数
			oldCount := lt.GetRecordCount()

			// 重载学习表
			if err := lt.load(); err != nil {
				fmt.Printf("重载学习表失败: %v\n", err)
			} else {
				newCount := lt.GetRecordCount()
				fmt.Printf("学习表重载成功! 记录数: %d -> %d\n", oldCount, newCount)
			}
		}
	}
}

func (lt *LearningTable) Close() error {
	// 轮询方式不需要特殊的关闭操作
	return nil
}

func (lt *LearningTable) GetRecordCount() int {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	return len(lt.records)
}
