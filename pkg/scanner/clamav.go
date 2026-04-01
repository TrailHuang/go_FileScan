package scanner

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type ClamAVResult struct {
	FilePath    string
	VirusName   string
	IsInfected  bool
	ScanTime    time.Time
	Error       string
}

type ClamAVScanner struct {
	socketPath string
	timeout    time.Duration
	conn       net.Conn
}

func NewClamAVScanner(socketPath string, timeout time.Duration) (*ClamAVScanner, error) {
	cas := &ClamAVScanner{
		socketPath: socketPath,
		timeout:    timeout,
	}

	if err := cas.connect(); err != nil {
		return nil, err
	}

	return cas, nil
}

func (cas *ClamAVScanner) connect() error {
	conn, err := net.DialTimeout("unix", cas.socketPath, cas.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to ClamAV socket: %w", err)
	}

	cas.conn = conn

	if err := cas.ping(); err != nil {
		cas.conn.Close()
		return fmt.Errorf("ClamAV ping failed: %w", err)
	}

	return nil
}

func (cas *ClamAVScanner) ping() error {
	if err := cas.conn.SetDeadline(time.Now().Add(cas.timeout)); err != nil {
		return err
	}

	_, err := cas.conn.Write([]byte("zPING\x00"))
	if err != nil {
		return err
	}

	response := make([]byte, 1024)
	n, err := cas.conn.Read(response)
	if err != nil {
		return err
	}

	if string(response[:n]) != "PONG\x00" {
		return fmt.Errorf("unexpected PING response: %s", string(response[:n]))
	}

	return nil
}

func (cas *ClamAVScanner) ScanFile(filePath string) (*ClamAVResult, error) {
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("file does not exist: %w", err)
	}

	if err := cas.conn.SetDeadline(time.Now().Add(cas.timeout)); err != nil {
		return nil, err
	}

	command := fmt.Sprintf("zSCAN %s\x00", filePath)
	_, err := cas.conn.Write([]byte(command))
	if err != nil {
		return nil, fmt.Errorf("failed to send SCAN command: %w", err)
	}

	response := make([]byte, 4096)
	n, err := cas.conn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("failed to read SCAN response: %w", err)
	}

	result := &ClamAVResult{
		FilePath: filePath,
		ScanTime: time.Now(),
	}

	return cas.parseResponse(string(response[:n]), result)
}

func (cas *ClamAVScanner) parseResponse(response string, result *ClamAVResult) (*ClamAVResult, error) {
	scanner := bufio.NewScanner(strings.NewReader(response))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, ": ")
		if len(parts) < 2 {
			continue
		}

		status := parts[1]

		if strings.HasSuffix(status, "FOUND") {
			virusParts := strings.Split(status, " ")
			if len(virusParts) >= 1 {
				result.VirusName = virusParts[0]
				result.IsInfected = true
			}
		} else if status == "OK" {
			result.IsInfected = false
		} else if strings.Contains(status, "ERROR") {
			result.Error = status
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("ClamAV error: %s", result.Error)
	}

	return result, nil
}

func (cas *ClamAVScanner) Close() error {
	if cas.conn != nil {
		return cas.conn.Close()
	}
	return nil
}

func (cas *ClamAVScanner) Reconnect() error {
	if cas.conn != nil {
		cas.conn.Close()
	}
	return cas.connect()
}