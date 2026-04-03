package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go-filescan/pkg/scanner"

	"github.com/fsnotify/fsnotify"
)

type DirectoryWatcher struct {
	watcher     *fsnotify.Watcher
	scanner     *scanner.FileScanner
	directories []string
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

func NewDirectoryWatcher(directories []string, fileScanner *scanner.FileScanner) (*DirectoryWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	dw := &DirectoryWatcher{
		watcher:     watcher,
		scanner:     fileScanner,
		directories: directories,
		stopChan:    make(chan struct{}),
	}

	for _, dir := range directories {
		if err := dw.addDirectory(dir); err != nil {
			watcher.Close()
			return nil, fmt.Errorf("failed to add directory %s: %w", dir, err)
		}
	}

	return dw, nil
}

func (dw *DirectoryWatcher) addDirectory(dirPath string) error {
	if _, err := os.Stat(dirPath); err != nil {
		return fmt.Errorf("directory does not exist: %w", err)
	}

	if err := dw.watcher.Add(dirPath); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	fmt.Printf("Watching directory: %s\n", dirPath)

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing path %s: %v\n", path, err)
			return nil
		}

		if info.IsDir() {
			if err := dw.watcher.Add(path); err != nil {
				fmt.Printf("Failed to watch subdirectory %s: %v\n", path, err)
			} else {
				fmt.Printf("Watching subdirectory: %s\n", path)
			}
		}

		return nil
	})
}

func (dw *DirectoryWatcher) Start(resultsChan chan<- *scanner.ScanResult) error {
	dw.wg.Add(1)
	go dw.watchLoop(resultsChan)

	return nil
}

func (dw *DirectoryWatcher) watchLoop(resultsChan chan<- *scanner.ScanResult) {
	defer dw.wg.Done()

	fmt.Println("Directory watcher loop started")

	for {
		select {
		case event, ok := <-dw.watcher.Events:
			if !ok {
				fmt.Println("Watcher events channel closed")
				return
			}

			if event.Has(fsnotify.Create) {
				dw.handleCreate(event.Name, resultsChan)
			} else if event.Has(fsnotify.Write) {
				dw.handleWrite(event.Name, resultsChan)
			} else if event.Has(fsnotify.Chmod) {
				dw.handleChmod(event.Name, resultsChan)
			} else if event.Has(fsnotify.Remove) {
				dw.handleRemove(event.Name)
			}

		case err, ok := <-dw.watcher.Errors:
			if !ok {
				fmt.Println("Watcher errors channel closed")
				return
			}
			fmt.Printf("Directory watcher error: %v\n", err)

		case <-dw.stopChan:
			fmt.Println("Directory watcher received stop signal")
			return
		}
	}
}

func (dw *DirectoryWatcher) handleCreate(path string, resultsChan chan<- *scanner.ScanResult) {
	fmt.Printf("Handling CREATE event for: %s\n", path)

	info, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Failed to stat created file %s: %v\n", path, err)
		return
	}

	if info.IsDir() {
		fmt.Printf("New directory created: %s\n", path)
		if err := dw.watcher.Add(path); err != nil {
			fmt.Printf("Failed to watch new directory %s: %v\n", path, err)
		} else {
			fmt.Printf("Now watching new directory: %s\n", path)
		}
		return
	}

	if !info.Mode().IsRegular() {
		fmt.Printf("Skipping non-regular file: %s\n", path)
		return
	}

	fmt.Printf("New file detected, scheduling scan: %s\n", path)
	time.Sleep(100 * time.Millisecond)

	go dw.scanFile(path, resultsChan)
}

func (dw *DirectoryWatcher) handleWrite(path string, resultsChan chan<- *scanner.ScanResult) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	if info.IsDir() || !info.Mode().IsRegular() {
		return
	}

	time.Sleep(100 * time.Millisecond)

	go dw.scanFile(path, resultsChan)
}

func (dw *DirectoryWatcher) handleChmod(path string, resultsChan chan<- *scanner.ScanResult) {
	fmt.Printf("Handling CHMOD event for: %s\n", path)

	info, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Failed to stat CHMOD file %s: %v\n", path, err)
		return
	}

	if info.IsDir() || !info.Mode().IsRegular() {
		fmt.Printf("Skipping CHMOD for non-regular file: %s\n", path)
		return
	}

	fmt.Printf("File permissions changed, scheduling scan: %s\n", path)
	time.Sleep(100 * time.Millisecond)

	go dw.scanFile(path, resultsChan)
}

func (dw *DirectoryWatcher) handleRemove(path string) {
	dw.watcher.Remove(path)
}

func (dw *DirectoryWatcher) scanFile(filePath string, resultsChan chan<- *scanner.ScanResult) {
	result, err := dw.scanner.ScanFile(filePath)
	if err != nil {
		result = &scanner.ScanResult{
			FilePath: filePath,
			Error:    err.Error(),
			ScanTime: time.Now(),
		}
	}

	select {
	case resultsChan <- result:
	case <-time.After(5 * time.Second):
		fmt.Printf("Timeout sending result for %s\n", filePath)
	case <-dw.stopChan:
		return
	}
}

func (dw *DirectoryWatcher) Stop() {
	close(dw.stopChan)
	if dw.watcher != nil {
		dw.watcher.Close()
	}
	dw.wg.Wait()
}

func (dw *DirectoryWatcher) InitialScan(resultsChan chan<- *scanner.ScanResult) error {
	for _, dir := range dw.directories {
		if err := dw.scanDirectory(dir, resultsChan); err != nil {
			return fmt.Errorf("failed to scan directory %s: %w", dir, err)
		}
	}
	return nil
}

func (dw *DirectoryWatcher) scanDirectory(dirPath string, resultsChan chan<- *scanner.ScanResult) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		select {
		case <-dw.stopChan:
			return filepath.SkipDir
		default:
		}

		if err != nil {
			return nil
		}

		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}

		go dw.scanFile(path, resultsChan)

		return nil
	})
}
