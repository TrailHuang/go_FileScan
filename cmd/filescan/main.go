package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"go-filescan/pkg/config"
	"go-filescan/pkg/learning"
	"go-filescan/pkg/output"
	"go-filescan/pkg/scanner"
	"go-filescan/pkg/watcher"
)

var (
	configPath  = flag.String("config", "config.yaml", "Path to configuration file")
	mode        = flag.String("mode", "watch", "Operation mode: watch, scan, or once")
	targetDir   = flag.String("dir", "", "Target directory to scan (overrides config)")
	outputFile  = flag.String("output", "", "Output file path (overrides config)")
	format      = flag.String("format", "", "Output format: json, text, csv (overrides config)")
	versionFlag = flag.Bool("version", false, "Show version information")
)

// 版本信息（通过编译时注入）
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Go文件病毒扫描程序\n")
		fmt.Printf("版本: %s\n", version)
		fmt.Printf("构建时间: %s\n", buildTime)
		fmt.Printf("Git提交: %s\n", gitCommit)
		fmt.Printf("Go版本: %s\n", runtime.Version())
		fmt.Printf("操作系统: %s\n", runtime.GOOS)
		fmt.Printf("架构: %s\n", runtime.GOARCH)
		return
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *targetDir != "" {
		cfg.Scanner.WatchDirectories = []string{*targetDir}
	}

	if *outputFile != "" {
		cfg.Output.File = *outputFile
	}

	if *format != "" {
		cfg.Output.Format = *format
	}

	outputConfig := output.OutputConfig{
		Format:            output.OutputFormat(cfg.Output.Format),
		File:              cfg.Output.File,
		IncludeCleanFiles: cfg.Output.IncludeCleanFiles,
	}

	resultWriter, err := output.NewResultWriter(outputConfig)
	if err != nil {
		log.Fatalf("Failed to create result writer: %v", err)
	}
	defer resultWriter.Close()

	learningTable, err := learning.NewLearningTable(cfg.Scanner.LearningTablePath, *mode)
	if err != nil {
		log.Fatalf("Failed to load learning table: %v", err)
	}
	defer learningTable.Close()

	fmt.Printf("Loaded learning table with %d records\n", learningTable.GetRecordCount())

	fileSizeLimit := parseFileSizeLimit(cfg.Scanner.Scan.FileSizeLimit)

	fileScanner, err := scanner.NewFileScanner(
		learningTable,
		cfg.Scanner.Quarantine,
		cfg.Scanner.Scan.MaxConcurrentScans,
		cfg.Scanner.Scan.ScanTimeout,
		fileSizeLimit,
	)
	if err != nil {
		log.Fatalf("Failed to create file scanner: %v", err)
	}
	defer fileScanner.Stop()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	switch *mode {
	case "watch":
		if err := runWatchMode(cfg, fileScanner, resultWriter, signalChan); err != nil {
			log.Fatalf("Watch mode failed: %v", err)
		}
	case "once":
		if err := runOnceMode(cfg, fileScanner, resultWriter); err != nil {
			log.Fatalf("Once mode failed: %v", err)
		}

	case "scan":
		if err := runScanMode(cfg, fileScanner, resultWriter, signalChan); err != nil {
			log.Fatalf("Scan mode failed: %v", err)
		}
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

// performScan 执行扫描并检测文件变化
func performScan(cfg *config.Config, fileScanner *scanner.FileScanner, resultWriter *output.ResultWriter, fileModTimes map[string]time.Time) error {
	// 获取目录下所有文件
	var filesToScan []string

	for _, dir := range cfg.Scanner.WatchDirectories {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			// 检查文件大小限制
			if cfg.Scanner.Scan.FileSizeLimit != "" {
				fileSizeLimit := parseFileSizeLimit(cfg.Scanner.Scan.FileSizeLimit)
				if fileSizeLimit > 0 && info.Size() > fileSizeLimit {
					fmt.Printf("Skipping large file: %s (%d bytes)\n", path, info.Size())
					return nil
				}
			}

			filesToScan = append(filesToScan, path)
			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk directory %s: %w", dir, err)
		}
	}

	var filesChanged []string

	// 检测文件变化
	for _, filePath := range filesToScan {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		currentModTime := fileInfo.ModTime()
		lastModTime, exists := fileModTimes[filePath]

		if !exists {
			// 新文件
			fmt.Printf("New file detected: %s\n", filePath)
			filesChanged = append(filesChanged, filePath)
			fileModTimes[filePath] = currentModTime
		} else if currentModTime.After(lastModTime) {
			// 文件已修改
			fmt.Printf("File modified: %s (last: %v, current: %v)\n", filePath, lastModTime, currentModTime)
			filesChanged = append(filesChanged, filePath)
			fileModTimes[filePath] = currentModTime
		}
	}

	// 扫描变化的文件
	if len(filesChanged) > 0 {
		fmt.Printf("Scanning %d changed files...\n", len(filesChanged))

		for _, filePath := range filesChanged {
			result, err := fileScanner.ScanFile(filePath)
			if err != nil {
				log.Printf("Failed to scan file %s: %v", filePath, err)
				continue
			}

			if err := resultWriter.WriteResult(result); err != nil {
				log.Printf("Failed to write result: %v", err)
			}
		}

		fmt.Printf("Completed scanning %d changed files\n", len(filesChanged))
	} else {
		fmt.Println("No file changes detected")
	}

	return nil
}

func runWatchMode(cfg *config.Config, fileScanner *scanner.FileScanner, resultWriter *output.ResultWriter, signalChan chan os.Signal) error {
	fmt.Println("Starting file system watcher...")

	dirWatcher, err := watcher.NewDirectoryWatcher(cfg.Scanner.WatchDirectories, fileScanner)
	if err != nil {
		return fmt.Errorf("failed to create directory watcher: %w", err)
	}
	defer dirWatcher.Stop()

	resultsChan := make(chan *scanner.ScanResult, 100)

	go func() {
		for result := range resultsChan {
			if err := resultWriter.WriteResult(result); err != nil {
				log.Printf("Failed to write result: %v", err)
			}
		}
	}()

	fmt.Println("Performing initial directory scan...")
	if err := dirWatcher.InitialScan(resultsChan); err != nil {
		return fmt.Errorf("initial scan failed: %w", err)
	}

	fmt.Println("Starting real-time monitoring...")
	if err := dirWatcher.Start(resultsChan); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	<-signalChan
	fmt.Println("\nReceived shutdown signal, stopping...")

	close(resultsChan)

	if err := resultWriter.WriteSummary(); err != nil {
		log.Printf("Failed to write summary: %v", err)
	}

	return nil
}

func runOnceMode(cfg *config.Config, fileScanner *scanner.FileScanner, resultWriter *output.ResultWriter) error {
	fmt.Println("Starting one-time directory scan...")

	resultsChan, err := fileScanner.ScanDirectory(cfg.Scanner.WatchDirectories[0])
	if err != nil {
		return fmt.Errorf("failed to start directory scan: %w", err)
	}

	for result := range resultsChan {
		if err := resultWriter.WriteResult(result); err != nil {
			log.Printf("Failed to write result: %v", err)
		}
	}

	if err := resultWriter.WriteSummary(); err != nil {
		log.Printf("Failed to write summary: %v", err)
	}

	fmt.Println("One-time scan completed, program exiting.")
	return nil
}

func runScanMode(cfg *config.Config, fileScanner *scanner.FileScanner, resultWriter *output.ResultWriter, signalChan chan os.Signal) error {
	fmt.Println("Starting periodic directory scan mode...")

	// 记录文件最后修改时间的映射
	fileModTimes := make(map[string]time.Time)

	// 扫描间隔（可配置，默认10秒）
	scanInterval := 10 * time.Second

	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	// 首次扫描
	fmt.Println("Performing initial scan...")
	if err := performScan(cfg, fileScanner, resultWriter, fileModTimes); err != nil {
		return fmt.Errorf("initial scan failed: %w", err)
	}

	fmt.Printf("Starting periodic scans every %v...\n", scanInterval)

	for {
		select {
		case <-ticker.C:
			fmt.Println("Starting periodic scan...")
			if err := performScan(cfg, fileScanner, resultWriter, fileModTimes); err != nil {
				log.Printf("Periodic scan failed: %v", err)
			}
		case <-signalChan:
			fmt.Println("\nReceived shutdown signal, stopping...")
			if err := resultWriter.WriteSummary(); err != nil {
				log.Printf("Failed to write summary: %v", err)
			}
			return nil
		}
	}
}

func parseFileSizeLimit(sizeStr string) int64 {
	if sizeStr == "" {
		return 0
	}

	var size int64
	var unit string

	n, _ := fmt.Sscanf(sizeStr, "%d%s", &size, &unit)
	if n < 1 {
		return 0
	}

	switch unit {
	case "B", "":
		return size
	case "KB":
		return size * 1024
	case "MB":
		return size * 1024 * 1024
	case "GB":
		return size * 1024 * 1024 * 1024
	default:
		return 0
	}
}
