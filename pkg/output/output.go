package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go-filescan/pkg/scanner"
)

type OutputFormat string

const (
	JSONFormat OutputFormat = "json"
	TextFormat OutputFormat = "text"
	CSVFormat  OutputFormat = "csv"
)

type OutputConfig struct {
	Format           OutputFormat
	File             string
	IncludeCleanFiles bool
}

type ResultWriter struct {
	config     OutputConfig
	file       *os.File
	mu         sync.Mutex
	totalFiles int
	infected   int
	clean      int
	errors     int
}

func NewResultWriter(config OutputConfig) (*ResultWriter, error) {
	rw := &ResultWriter{
		config: config,
	}

	if config.File != "" {
		if err := os.MkdirAll(filepath.Dir(config.File), 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}

		file, err := os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open output file: %w", err)
		}
		rw.file = file
	}

	return rw, nil
}

func (rw *ResultWriter) WriteResult(result *scanner.ScanResult) error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	rw.totalFiles++

	if result.Error != "" {
		rw.errors++
	} else if result.IsInfected {
		rw.infected++
	} else {
		rw.clean++
	}

	var output string
	switch rw.config.Format {
	case JSONFormat:
		jsonData, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		output = string(jsonData) + "\n"
	case TextFormat:
		output = rw.formatText(result)
	case CSVFormat:
		output = rw.formatCSV(result)
	default:
		output = rw.formatText(result)
	}

	if rw.file != nil {
		if _, err := rw.file.WriteString(output); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	} else {
		fmt.Print(output)
	}

	return nil
}

func (rw *ResultWriter) formatText(result *scanner.ScanResult) string {

	output := fmt.Sprintf("%s - %s", result.ScanTime.Format("2006-01-02 15:04:05"), result.FilePath)

	if result.IsInfected {
		output += fmt.Sprintf(" - Virus: %s", result.VirusName)
	}

	if result.Error != "" {
		output += fmt.Sprintf(" - Error: %s", result.Error)
	}

	if result.MD5 != "" {
		output += fmt.Sprintf(" - MD5: %s", result.MD5)
	}

	return output + "\n"
}

func (rw *ResultWriter) formatCSV(result *scanner.ScanResult) string {

	return fmt.Sprintf("%s,%s,%s,%s,%s\n",
		result.ScanTime.Format("2006-01-02 15:04:05"),
		result.FilePath,
		result.VirusName,
		result.MD5,
		result.Error)
}

func (rw *ResultWriter) WriteSummary() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	summary := fmt.Sprintf("\n=== Scan Summary ===\n"+
		"Total files scanned: %d\n"+
		"Infected files: %d\n"+
		"Clean files: %d\n"+
		"Errors: %d\n"+
		"Scan completed at: %s\n",
		rw.totalFiles, rw.infected, rw.clean, rw.errors, time.Now().Format("2006-01-02 15:04:05"))

	if rw.file != nil {
		if _, err := rw.file.WriteString(summary); err != nil {
			return fmt.Errorf("failed to write summary: %w", err)
		}
	} else {
		fmt.Print(summary)
	}

	return nil
}

func (rw *ResultWriter) Close() error {
	if rw.file != nil {
		return rw.file.Close()
	}
	return nil
}

func (rw *ResultWriter) GetStats() (total, infected, clean, errors int) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	return rw.totalFiles, rw.infected, rw.clean, rw.errors
}