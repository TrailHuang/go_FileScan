package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go-filescan/pkg/scanner"
)

type WebServer struct {
	scanner     *scanner.FileScanner
	port        int
	samplesDir  string
	scanResults []*scanner.ScanResult
	watchDir    string
}

func NewWebServer(scanner *scanner.FileScanner, port int, samplesDir string, watchDir string) *WebServer {
	return &WebServer{
		scanner:    scanner,
		port:       port,
		samplesDir: samplesDir,
		watchDir:   watchDir,
	}
}

func (ws *WebServer) Start() error {
	http.HandleFunc("/", ws.handleRoot)
	http.HandleFunc("/upload", ws.handleUpload)
	http.HandleFunc("/scan", ws.handleScan)
	http.HandleFunc("/results", ws.handleResults)

	addr := fmt.Sprintf(":%d", ws.port)
	fmt.Printf("Web 服务器启动在端口 %d\n", ws.port)
	fmt.Printf("访问 http://localhost:%d 查看上传页面\n", ws.port)

	return http.ListenAndServe(addr, nil)
}

func (ws *WebServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>文件病毒扫描</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
        }
        .header {
            text-align: center;
            color: white;
            margin-bottom: 40px;
        }
        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }
        .card {
            background: white;
            border-radius: 16px;
            padding: 30px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
        }
        .form-group {
            margin-bottom: 20px;
        }
        .form-group label {
            display: block;
            margin-bottom: 8px;
            color: #333;
            font-weight: 600;
        }
        .form-group input[type="file"] {
            width: 100%;
            padding: 12px;
            border: 2px dashed #667eea;
            border-radius: 8px;
            background: #f8f9ff;
            cursor: pointer;
        }
        .form-group input[type="file"]:hover {
            background: #f0f1ff;
        }
        .btn {
            width: 100%;
            padding: 14px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 1.1em;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s;
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 20px rgba(102, 126, 234, 0.4);
        }
        .btn:disabled {
            opacity: 0.6;
            cursor: not-allowed;
        }
        .result {
            margin-top: 20px;
            padding: 20px;
            border-radius: 8px;
            background: #f8f9ff;
        }
        .result.virus {
            background: #fff5f5;
            border: 2px solid #fc8181;
        }
        .result.clean {
            background: #f0fff4;
            border: 2px solid #68d391;
        }
        .result h3 {
            margin-bottom: 10px;
            color: #333;
        }
        .result p {
            margin: 5px 0;
            color: #666;
        }
        .result .virus-name {
            color: #e53e3e;
            font-weight: 600;
        }
        .result .clean-name {
            color: #38a169;
            font-weight: 600;
        }
        .loading {
            display: none;
            text-align: center;
            color: #667eea;
            margin-top: 20px;
        }
        .loading.show {
            display: block;
        }
        .spinner {
            border: 3px solid #f3f3f3;
            border-top: 3px solid #667eea;
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 0 auto 10px;
        }
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        .history {
            margin-top: 30px;
        }
        .history h3 {
            margin-bottom: 15px;
            color: #333;
        }
        .history-item {
            padding: 15px;
            background: #f8f9ff;
            border-radius: 8px;
            margin-bottom: 10px;
        }
        .history-item p {
            margin: 5px 0;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🛡️ 文件病毒扫描系统</h1>
            <p>上传文件进行安全扫描</p>
        </div>

        <div class="card">
            <form id="uploadForm" enctype="multipart/form-data">
                <div class="form-group">
                    <label for="file">选择文件：</label>
                    <input type="file" id="file" name="file" required>
                </div>
                <button type="submit" class="btn" id="submitBtn">上传并扫描</button>
            </form>

            <div class="loading" id="loading">
                <div class="spinner"></div>
                <p>正在扫描，请稍候...</p>
            </div>

            <div id="result"></div>
        </div>

        <div class="history">
            <h3>最近扫描记录</h3>
            <div id="history"></div>
        </div>
    </div>

    <script>
        let scanResults = [];

        document.getElementById('uploadForm').addEventListener('submit', async (e) => {
            e.preventDefault();

            const fileInput = document.getElementById('file');
            const submitBtn = document.getElementById('submitBtn');
            const loading = document.getElementById('loading');
            const resultDiv = document.getElementById('result');

            if (!fileInput.files.length) {
                alert('请选择文件');
                return;
            }

            const file = fileInput.files[0];
            submitBtn.disabled = true;
            submitBtn.textContent = '上传中...';
            loading.classList.add('show');
            resultDiv.innerHTML = '';

            const formData = new FormData();
            formData.append('file', file);

            try {
                const response = await fetch('/upload', {
                    method: 'POST',
                    body: formData
                });

                if (!response.ok) {
                    throw new Error('上传失败');
                }

                const result = await response.json();
                displayResult(result);
                addToHistory(result);

                // 清空文件输入
                fileInput.value = '';
            } catch (error) {
                resultDiv.innerHTML = '<div class="result"><h3>❌ 扫描失败</h3><p>' + error.message + '</p></div>';
            } finally {
                submitBtn.disabled = false;
                submitBtn.textContent = '上传并扫描';
                loading.classList.remove('show');
            }
        });

        function displayResult(result) {
            const resultDiv = document.getElementById('result');

            if (result.IsInfected) {
                resultDiv.innerHTML = '<div class="result virus">' +
                    '<h3>⚠️ 检测到病毒</h3>' +
                    '<p><strong>文件名：</strong> ' + result.FileName + '</p>' +
                    '<p><strong>病毒名称：</strong> <span class="virus-name">' + result.VirusName + '</span></p>' +
                    '<p><strong>扫描方法：</strong> ' + result.ScanMethod + '</p>' +
                    '<p><strong>扫描时间：</strong> ' + new Date(result.ScanTime).toLocaleString('zh-CN') + '</p>' +
                    (result.Error ? '<p><strong>错误信息：</strong> ' + result.Error + '</p>' : '') +
                    '</div>';
            } else {
                resultDiv.innerHTML = '<div class="result clean">' +
                    '<h3>✅ 安全文件</h3>' +
                    '<p><strong>文件名：</strong> ' + result.FileName + '</p>' +
                    '<p><strong>状态：</strong> <span class="clean-name">白样本</span></p>' +
                    '<p><strong>扫描方法：</strong> ' + result.ScanMethod + '</p>' +
                    '<p><strong>扫描时间：</strong> ' + new Date(result.ScanTime).toLocaleString('zh-CN') + '</p>' +
                    '</div>';
            }
        }

        function addToHistory(result) {
            scanResults.unshift(result);
            if (scanResults.length > 10) {
                scanResults.pop();
            }
            renderHistory();
        }

        function renderHistory() {
            const historyDiv = document.getElementById('history');
            if (scanResults.length === 0) {
                historyDiv.innerHTML = '<p style="color: #999;">暂无扫描记录</p>';
                return;
            }

            historyDiv.innerHTML = scanResults.map((result, index) => {
                return '<div class="history-item">' +
                    '<p><strong>#' + (index + 1) + ' ' + result.FileName + '</strong></p>' +
                    '<p style="color: ' + (result.IsInfected ? '#e53e3e' : '#38a169') + '">' +
                    (result.IsInfected ? '⚠️ 检测到病毒: ' + result.VirusName : '✅ 安全') +
                    '</p>' +
                    '<p style="color: #999; font-size: 0.85em;">' +
                    new Date(result.ScanTime).toLocaleString('zh-CN') +
                    '</p>' +
                    '</div>';
            }).join('');
        }

        // 初始化时加载历史记录
        fetch('/results')
            .then(res => res.json())
            .then(data => {
                scanResults = data;
                renderHistory();
            })
            .catch(() => {});
    </script>
</body>
</html>`)
}

func (ws *WebServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 保存文件到配置的监控目录
	timestamp := time.Now().UnixNano()
	safeName := filepath.Base(header.Filename)
	savedPath := filepath.Join(ws.watchDir, fmt.Sprintf("%d_%s", timestamp, safeName))

	out, err := os.Create(savedPath)
	if err != nil {
		http.Error(w, "Failed to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		http.Error(w, "Failed to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 扫描文件
	result, err := ws.scanner.ScanFile(savedPath)
	if err != nil {
		result = &scanner.ScanResult{
			FilePath: savedPath,
			ScanTime: time.Now(),
			Error:    "Scan failed: " + err.Error(),
		}
	}

	// 添加到扫描历史记录
	ws.scanResults = append([]*scanner.ScanResult{result}, ws.scanResults...)
	if len(ws.scanResults) > 10 {
		ws.scanResults = ws.scanResults[:10]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (ws *WebServer) handleScan(w http.ResponseWriter, r *http.Request) {
	// 用于直接扫描接口（保留）
	http.Error(w, "Use /upload for scanning", http.StatusMethodNotAllowed)
}

func (ws *WebServer) handleResults(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ws.scanResults)
}
