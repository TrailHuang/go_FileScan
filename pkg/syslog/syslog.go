package syslog

import (
	"fmt"
	"net"
	"os"
	"time"

	"go-filescan/pkg/config"
)

// Severity 日志级别
type Severity int

const (
	SeverityEmergency Severity = 0
	SeverityAlert     Severity = 1
	SeverityCritical  Severity = 2
	SeverityError     Severity = 3
	SeverityWarning   Severity = 4
	SeverityNotice    Severity = 5
	SeverityInfo      Severity = 6
	SeverityDebug     Severity = 7
)

// Facility 设备类型
type Facility int

const (
	FacilityUser   Facility = 1  // user-level messages
	FacilityDaemon Facility = 3  // system daemons
	FacilityLocal0 Facility = 16 // local use 0
	FacilityLocal1 Facility = 17
	FacilityLocal2 Facility = 18
	FacilityLocal3 Facility = 19
	FacilityLocal4 Facility = 20
	FacilityLocal5 Facility = 21
	FacilityLocal6 Facility = 22
	FacilityLocal7 Facility = 23
)

// Sender syslog 发送器
type Sender struct {
	config   config.SyslogConfig
	hostname string
	conn     net.Conn
}

// NewSender 创建 syslog 发送器
func NewSender(cfg config.SyslogConfig) (*Sender, error) {
	s := &Sender{
		config: cfg,
	}

	var err error
	s.hostname, err = os.Hostname()
	if err != nil {
		s.hostname = "unknown"
	}

	if cfg.Enabled {
		addr := fmt.Sprintf("%s:%d", cfg.Server, cfg.Port)
		conn, err := net.DialTimeout("udp", addr, 3*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to syslog server %s: %w", addr, err)
		}
		s.conn = conn
	}

	return s, nil
}

// Send 发送 syslog 消息
func (s *Sender) Send(severity Severity, message string) error {
	if !s.config.Enabled || s.conn == nil {
		return nil
	}

	facility := parseFacility(s.config.Facility)
	priority := int(facility)*8 + int(severity)
	tag := s.config.Tag
	if tag == "" {
		tag = "FileScanner"
	}

	// RFC 3164 格式: <PRI>TIMESTAMP HOSTNAME TAG: MESSAGE
	timestamp := time.Now().Format("Jan _2 15:04:05")
	syslogMsg := fmt.Sprintf("<%d>%s %s %s: %s",
		priority, timestamp, s.hostname, tag, message)

	_, err := s.conn.Write([]byte(syslogMsg))
	if err != nil {
		return fmt.Errorf("failed to send syslog message: %w", err)
	}

	return nil
}

// SendAlert 发送告警级别的 syslog 消息（病毒检测）
func (s *Sender) SendAlert(filePath, md5, virusName, scanMethod string, scanTime time.Time) error {
	message := fmt.Sprintf("VIRUS DETECTED | File: %s | MD5: %s | Virus: %s | Method: %s | Time: %s",
		filePath, md5, virusName, scanMethod, scanTime.Format("2006-01-02 15:04:05"))
	return s.Send(SeverityAlert, message)
}

// SendInfo 发送信息级别的 syslog 消息
func (s *Sender) SendInfo(message string) error {
	return s.Send(SeverityInfo, message)
}

// Close 关闭连接
func (s *Sender) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// parseFacility 解析 Facility 字符串
func parseFacility(facility string) Facility {
	switch facility {
	case "user":
		return FacilityUser
	case "daemon":
		return FacilityDaemon
	case "local0":
		return FacilityLocal0
	case "local1":
		return FacilityLocal1
	case "local2":
		return FacilityLocal2
	case "local3":
		return FacilityLocal3
	case "local4":
		return FacilityLocal4
	case "local5":
		return FacilityLocal5
	case "local6":
		return FacilityLocal6
	case "local7":
		return FacilityLocal7
	default:
		return FacilityDaemon
	}
}
