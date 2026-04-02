package scanner

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-filescan/pkg/learning"
)

type ScanResult struct {
	FilePath   string    `json:"file_path"`
	MD5        string    `json:"md5"`
	VirusName  string    `json:"virus_name"`
	IsInfected bool      `json:"-"`
	ScanMethod string    `json:"-"`
	ScanTime   time.Time `json:"scan_time"`
	Error      string    `json:"error,omitempty"`
}

type FileScanner struct {
	learningTable *learning.LearningTable
	maxWorkers    int
	scanTimeout   time.Duration
	fileSizeLimit int64
	resultsChan   chan *ScanResult
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

func NewFileScanner(learningTable *learning.LearningTable, maxWorkers int, scanTimeout time.Duration, fileSizeLimit int64) (*FileScanner, error) {
	fs := &FileScanner{
		learningTable: learningTable,
		maxWorkers:    maxWorkers,
		scanTimeout:   scanTimeout,
		fileSizeLimit: fileSizeLimit,
		resultsChan:   make(chan *ScanResult, 100),
		stopChan:      make(chan struct{}),
	}

	return fs, nil
}

func (fs *FileScanner) ScanFile(filePath string) (*ScanResult, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	if fs.fileSizeLimit > 0 && fileInfo.Size() > fs.fileSizeLimit {
		return nil, fmt.Errorf("file size %d exceeds limit %d", fileInfo.Size(), fs.fileSizeLimit)
	}

	md5Hash, err := fs.calculateMD5(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate MD5: %w", err)
	}

	fmt.Printf("Scanning file: %s, MD5: %s\n", filePath, md5Hash)

	result := &ScanResult{
		FilePath: filePath,
		MD5:      md5Hash,
		ScanTime: time.Now(),
	}

	if record, exists := fs.learningTable.Lookup(strings.ToUpper(md5Hash)); exists {
		fmt.Printf("VIRUS DETECTED! File: %s, Virus: %s\n", filePath, record.VirusName)
		result.VirusName = record.VirusName
		result.IsInfected = true
		result.ScanMethod = "learning_table"
		return result, nil
	}

	fmt.Printf("No virus found in learning table for MD5: %s\n", md5Hash)
	result.VirusName = "白样本"
	result.IsInfected = false
	result.ScanMethod = "md5_only"

	return result, nil
}

func (fs *FileScanner) calculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (fs *FileScanner) ScanDirectory(dirPath string) (<-chan *ScanResult, error) {
	if _, err := os.Stat(dirPath); err != nil {
		return nil, fmt.Errorf("directory does not exist: %w", err)
	}

	workChan := make(chan string, fs.maxWorkers*2)
	fs.resultsChan = make(chan *ScanResult, 100)

	for i := 0; i < fs.maxWorkers; i++ {
		fs.wg.Add(1)
		go fs.worker(workChan)
	}

	go fs.walkDirectory(dirPath, workChan)

	return fs.resultsChan, nil
}

func (fs *FileScanner) walkDirectory(dirPath string, workChan chan<- string) {
	defer close(workChan)

	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		select {
		case <-fs.stopChan:
			return filepath.SkipDir
		default:
		}

		if err != nil {
			return nil
		}

		if !info.IsDir() && !info.Mode().IsRegular() {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		select {
		case workChan <- path:
		case <-fs.stopChan:
			return filepath.SkipDir
		}

		return nil
	})

	fs.wg.Wait()
	close(fs.resultsChan)
}

func (fs *FileScanner) worker(workChan <-chan string) {
	defer fs.wg.Done()

	for filePath := range workChan {
		select {
		case <-fs.stopChan:
			return
		default:
		}

		result, err := fs.ScanFile(filePath)
		if err != nil {
			result = &ScanResult{
				FilePath: filePath,
				Error:    err.Error(),
				ScanTime: time.Now(),
			}
		}

		select {
		case fs.resultsChan <- result:
		case <-fs.stopChan:
			return
		}
	}
}

func (fs *FileScanner) Stop() {
	close(fs.stopChan)
	fs.wg.Wait()
}

func (fs *FileScanner) GetResultsChannel() <-chan *ScanResult {
	return fs.resultsChan
}
