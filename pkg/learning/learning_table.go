package learning

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type VirusRecord struct {
	MD5      string
	Size     string
	VirusName string
}

type LearningTable struct {
	filePath string
	records  map[string]*VirusRecord
	mu       sync.RWMutex
	watcher  *fsnotify.Watcher
}

func NewLearningTable(filePath string) (*LearningTable, error) {
	lt := &LearningTable{
		filePath: filePath,
		records:  make(map[string]*VirusRecord),
	}

	if err := lt.load(); err != nil {
		return nil, err
	}

	if err := lt.startWatching(); err != nil {
		return nil, err
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
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	lt.watcher = watcher

	if err := lt.watcher.Add(lt.filePath); err != nil {
		return fmt.Errorf("failed to watch learning table file: %w", err)
	}

	go lt.watchChanges()

	return nil
}

func (lt *LearningTable) watchChanges() {
	for {
		select {
		case event, ok := <-lt.watcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				time.Sleep(100 * time.Millisecond)
				if err := lt.load(); err != nil {
					fmt.Printf("Failed to reload learning table: %v\n", err)
				} else {
					fmt.Println("Learning table reloaded successfully")
				}
			}

		case err, ok := <-lt.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Learning table watcher error: %v\n", err)
		}
	}
}

func (lt *LearningTable) Close() error {
	if lt.watcher != nil {
		return lt.watcher.Close()
	}
	return nil
}

func (lt *LearningTable) GetRecordCount() int {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	return len(lt.records)
}