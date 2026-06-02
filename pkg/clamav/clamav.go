package clamav

/*
#cgo LDFLAGS: -lclamav
#include <clamav.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

// Config ClamAV 嵌入式引擎配置
type Config struct {
	Enabled     bool   `mapstructure:"enabled"`      // 是否启用 ClamAV 扫描
	DatabaseDir string `mapstructure:"database_dir"` // 病毒特征库目录（如 /var/lib/clamav/）
}

// Scanner ClamAV 嵌入式扫描器（通过 libclamav 直接调用，无需 clamd 服务）
type Scanner struct {
	config  Config
	engine  *C.struct_cl_engine
	mu      sync.Mutex // 保护扫描并发安全
	closed  bool
}

// NewScanner 创建 ClamAV 嵌入式扫描器，初始化引擎并加载病毒特征库
func NewScanner(config Config) (*Scanner, error) {
	if !config.Enabled {
		return &Scanner{config: config}, nil
	}

	if config.DatabaseDir == "" {
		config.DatabaseDir = "/var/lib/clamav"
	}

	// 初始化 ClamAV 库
	if ret := C.cl_init(C.CL_INIT_DEFAULT); ret != C.CL_SUCCESS {
		return nil, fmt.Errorf("cl_init failed with code %d", int(ret))
	}

	// 创建扫描引擎
	engine := C.cl_engine_new()
	if engine == nil {
		return nil, fmt.Errorf("cl_engine_new returned NULL")
	}

	// 加载病毒特征库
	dbDir := C.CString(config.DatabaseDir)
	defer C.free(unsafe.Pointer(dbDir))

	var sigCount C.uint
	ret := C.cl_load(dbDir, engine, &sigCount, C.CL_DB_STDOPT)
	if ret != C.CL_SUCCESS {
		C.cl_engine_free(engine)
		return nil, fmt.Errorf("cl_load failed from %s (code %d)", config.DatabaseDir, int(ret))
	}

	// 编译引擎（加载后必须编译）
	if ret := C.cl_engine_compile(engine); ret != C.CL_SUCCESS {
		C.cl_engine_free(engine)
		return nil, fmt.Errorf("cl_engine_compile failed with code %d", int(ret))
	}

	fmt.Printf("ClamAV engine initialized, loaded %d signatures from %s\n", int(sigCount), config.DatabaseDir)

	return &Scanner{
		config: config,
		engine: engine,
	}, nil
}

// Enabled 返回 ClamAV 是否启用且引擎正常
func (s *Scanner) Enabled() bool {
	return s.config.Enabled && s.engine != nil && !s.closed
}

// ScanFile 使用嵌入式 ClamAV 引擎扫描文件
// 返回: 病毒名称, 是否感染, 错误
func (s *Scanner) ScanFile(filePath string) (string, bool, error) {
	if !s.Enabled() {
		return "", false, fmt.Errorf("clamav engine not available")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.engine == nil {
		return "", false, fmt.Errorf("clamav engine is closed")
	}

	cPath := C.CString(filePath)
	defer C.free(unsafe.Pointer(cPath))

	var virname *C.char
	var scanned C.ulong
	var scanOpts C.struct_cl_scan_options // 零值 = 默认扫描选项

	ret := C.cl_scanfile(cPath, &virname, &scanned, s.engine, &scanOpts)

	switch ret {
	case C.CL_CLEAN:
		return "", false, nil
	case C.CL_VIRUS:
		virusName := C.GoString(virname)
		return virusName, true, nil
	default:
		return "", false, fmt.Errorf("cl_scanfile failed with code %d", int(ret))
	}
}

// Close 释放 ClamAV 引擎资源
func (s *Scanner) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.engine == nil {
		return nil
	}

	ret := C.cl_engine_free(s.engine)
	s.engine = nil
	s.closed = true

	if ret != C.CL_SUCCESS {
		return fmt.Errorf("cl_engine_free failed with code %d", int(ret))
	}

	fmt.Println("ClamAV engine closed")
	return nil
}
