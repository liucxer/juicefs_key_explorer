package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

// KeyInfo 存储解析后的 key 信息
type KeyInfo struct {
	UUID         string                 `json:"uuid"`
	Type         string                 `json:"type"`
	SubType      string                 `json:"subType"`
	Description  string                 `json:"description"`
	Details      map[string]interface{} `json:"details"`
	ValueDetails map[string]interface{} `json:"valueDetails"`
}

// ValueInfo 存储解析后的 value 信息
type ValueInfo struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Value       interface{}            `json:"value"`
	Details     map[string]interface{} `json:"details"`
}

// ScanResult 扫描结果
type ScanResult struct {
	Key       string     `json:"key"`
	KeyInfo   *KeyInfo   `json:"keyInfo"`
	ValueInfo *ValueInfo `json:"valueInfo"`
}

// ScanRequest 扫描请求
type ScanRequest struct {
	PdAddr     string `json:"pdAddr"`
	CaPath     string `json:"caPath"`
	CertPath   string `json:"certPath"`
	KeyPath    string `json:"keyPath"`
	TypeFilter string `json:"typeFilter"`
}

// 生成模拟数据
func generateMockData(typeFilter string) []ScanResult {
	mockData := []ScanResult{
		{
			Key: "juicefs-fslist-prefix\\xfdFSIFuuid-12345",
			KeyInfo: &KeyInfo{
				UUID:        "",
				Type:        "FileSystem",
				SubType:     "",
				Description: "JuiceFS file system list",
				Details: map[string]interface{}{
					"FSInfo": "uuid-12345",
				},
				ValueDetails: map[string]interface{}{},
			},
			ValueInfo: &ValueInfo{
				Type:        "Unknown",
				Description: "Unknown value type",
				Value:       "0x01",
				Details:     map[string]interface{}{},
			},
		},
		{
			Key: "uuid-12345\\xfdA\\x00\\x00\\x00\\x00\\x00\\x00\\x00\\x01I",
			KeyInfo: &KeyInfo{
				UUID:        "uuid-12345",
				Type:        "A",
				SubType:     "I",
				Description: "Inode attribute",
				Details: map[string]interface{}{
					"Inode": 1,
				},
				ValueDetails: map[string]interface{}{},
			},
			ValueInfo: &ValueInfo{
				Type:        "Node",
				Description: "Binary encoded file attributes",
				Value:       "0x000081a400000000000000005f3c1c9c000000005f3c1c9c000000005f3c1c9c0000000000000001000000000000000000000000",
				Details: map[string]interface{}{
					"Flags":     0,
					"Mode":      40964,
					"Uid":       0,
					"Gid":       0,
					"Atime":     1600000000,
					"Mtime":     1600000000,
					"Ctime":     1600000000,
					"Nlink":     1,
					"Length":    0,
					"Rdev":      0,
					"TypeName":  "Directory",
					"AtimeStr":  time.Unix(1600000000, 0).Format(time.RFC3339),
					"MtimeStr":  time.Unix(1600000000, 0).Format(time.RFC3339),
					"CtimeStr":  time.Unix(1600000000, 0).Format(time.RFC3339),
				},
			},
		},
		{
			Key: "uuid-12345\\xfdCnextInode",
			KeyInfo: &KeyInfo{
				UUID:        "uuid-12345",
				Type:        "C",
				SubType:     "",
				Description: "Next inode number",
				Details:     map[string]interface{}{},
				ValueDetails: map[string]interface{}{},
			},
			ValueInfo: &ValueInfo{
				Type:        "Counter",
				Description: "Integer value",
				Value:       1000,
				Details: map[string]interface{}{
					"Value": 1000,
				},
			},
		},
		{
			Key: "uuid-12345\\xfdSI\\x00\\x00\\x00\\x00\\x00\\x00\\x00\\x01",
			KeyInfo: &KeyInfo{
				UUID:        "uuid-12345",
				Type:        "SI",
				SubType:     "",
				Description: "Session info",
				Details: map[string]interface{}{
					"SessionID": 1,
				},
				ValueDetails: map[string]interface{}{},
			},
			ValueInfo: &ValueInfo{
				Type:        "SessionInfo",
				Description: "JSON format session information",
				Value:       `{"client":"juicefs", "version":"1.0.0"}`,
				Details: map[string]interface{}{
					"client":  "juicefs",
					"version": "1.0.0",
				},
			},
		},
		{
			Key: "setting",
			KeyInfo: &KeyInfo{
				UUID:        "",
				Type:        "Config",
				SubType:     "",
				Description: "JuiceFS configuration",
				Details:     map[string]interface{}{},
				ValueDetails: map[string]interface{}{},
			},
			ValueInfo: &ValueInfo{
				Type:        "Setting",
				Description: "JSON format filesystem configuration",
				Value:       `{"name":"test-fs", "storage":"s3", "bucket":"juicefs-test"}`,
				Details: map[string]interface{}{
					"name":    "test-fs",
					"storage": "s3",
					"bucket":  "juicefs-test",
				},
			},
		},
	}

	// 应用类型筛选
	if typeFilter != "" {
		var filtered []ScanResult
		for _, result := range mockData {
			if result.KeyInfo.Type == typeFilter {
				filtered = append(filtered, result)
			}
		}
		return filtered
	}

	return mockData
}

// 处理 CORS
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// 处理扫描请求
func scanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 生成模拟数据
	results := generateMockData(req.TypeFilter)

	response := map[string]interface{}{
		"results": results,
		"count":   len(results),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 健康检查
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// 处理前端页面
func frontendHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>TiKV Key 扫描与解析工具</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: Arial, sans-serif;
            background-color: #f5f5f5;
            color: #333;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        
        h1 {
            text-align: center;
            margin-bottom: 30px;
            color: #2c3e50;
        }
        
        .form-container {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        
        .form-group {
            margin-bottom: 15px;
        }
        
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        
        input[type="text"] {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 16px;
        }
        
        .button-group {
            display: flex;
            gap: 10px;
            margin-top: 20px;
        }
        
        button {
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            font-size: 16px;
            cursor: pointer;
        }
        
        .btn-primary {
            background-color: #3498db;
            color: white;
        }
        
        .btn-primary:hover {
            background-color: #2980b9;
        }
        
        .btn-secondary {
            background-color: #95a5a6;
            color: white;
        }
        
        .btn-secondary:hover {
            background-color: #7f8c8d;
        }
        
        .filter-container {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        
        select {
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 16px;
            width: 200px;
        }
        
        .table-container {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            overflow-x: auto;
        }
        
        table {
            width: 100%;
            border-collapse: collapse;
        }
        
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        
        th {
            background-color: #f2f2f2;
            font-weight: bold;
        }
        
        tr:hover {
            background-color: #f5f5f5;
        }
        
        .status {
            margin-top: 20px;
            padding: 10px;
            border-radius: 4px;
        }
        
        .status-success {
            background-color: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        
        .status-error {
            background-color: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
        
        .status-info {
            background-color: #d1ecf1;
            color: #0c5460;
            border: 1px solid #bee5eb;
        }
        
        .loading {
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 3px solid rgba(52, 152, 219, 0.3);
            border-radius: 50%;
            border-top-color: #3498db;
            animation: spin 1s ease-in-out infinite;
        }
        
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        
        .details {
            margin-top: 5px;
            padding: 10px;
            background-color: #f9f9f9;
            border-radius: 4px;
            font-size: 14px;
        }
        
        .details pre {
            white-space: pre-wrap;
            word-break: break-all;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>TiKV Key 扫描与解析工具</h1>
        
        <!-- 表单容器 -->
        <div class="form-container">
            <h2>连接配置</h2>
            <div class="form-group">
                <label for="pdAddr">PD 地址</label>
                <input type="text" id="pdAddr" placeholder="例如: 192.168.210.210:2379" value="192.168.210.210:2379">
            </div>
            <div class="form-group">
                <label for="caPath">CA 证书路径 (可选)</label>
                <input type="text" id="caPath" placeholder="例如: /path/to/ca.crt">
            </div>
            <div class="form-group">
                <label for="certPath">客户端证书路径 (可选)</label>
                <input type="text" id="certPath" placeholder="例如: /path/to/client.crt">
            </div>
            <div class="form-group">
                <label for="keyPath">客户端密钥路径 (可选)</label>
                <input type="text" id="keyPath" placeholder="例如: /path/to/client.key">
            </div>
            <div class="button-group">
                <button class="btn-primary" onclick="scanTiKV()">开始扫描</button>
                <button class="btn-secondary" onclick="clearResults()">清空结果</button>
            </div>
        </div>
        
        <!-- 筛选容器 -->
        <div class="filter-container">
            <h2>筛选条件</h2>
            <div class="form-group">
                <label for="typeFilter">数据类型</label>
                <select id="typeFilter" onchange="applyFilter()">
                    <option value="">全部类型</option>
                    <option value="A">A (Inode 相关)</option>
                    <option value="C">C (计数器)</option>
                    <option value="S">S (会话相关)</option>
                    <option value="SE">SE (会话过期时间)</option>
                    <option value="SI">SI (会话信息)</option>
                    <option value="SH">SH (会话心跳)</option>
                    <option value="SS">SS (持续 inode)</option>
                    <option value="Config">Config (配置)</option>
                    <option value="FileSystem">FileSystem (文件系统)</option>
                    <option value="Unknown">Unknown (未知)</option>
                </select>
            </div>
        </div>
        
        <!-- 状态信息 -->
        <div id="status" class="status" style="display: none;"></div>
        
        <!-- 表格容器 -->
        <div class="table-container">
            <h2>扫描结果</h2>
            <table id="resultsTable">
                <thead>
                    <tr>
                        <th>类型</th>
                        <th>Key</th>
                        <th>Value</th>
                        <th>详细信息</th>
                    </tr>
                </thead>
                <tbody id="resultsBody">
                    <tr>
                        <td colspan="4" style="text-align: center; padding: 20px;">请点击"开始扫描"按钮开始扫描 TiKV</td>
                    </tr>
                </tbody>
            </table>
        </div>
    </div>
    
    <script>
        let allResults = [];
        
        // 扫描 TiKV
        function scanTiKV() {
            const pdAddr = document.getElementById('pdAddr').value;
            const caPath = document.getElementById('caPath').value;
            const certPath = document.getElementById('certPath').value;
            const keyPath = document.getElementById('keyPath').value;
            const typeFilter = document.getElementById('typeFilter').value;
            
            if (!pdAddr) {
                showStatus('请输入 PD 地址', 'error');
                return;
            }
            
            showStatus('正在扫描 TiKV... <div class="loading"></div>', 'info');
            
            // 发送请求到后端 API
            fetch('/api/scan', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    pdAddr: pdAddr,
                    caPath: caPath,
                    certPath: certPath,
                    keyPath: keyPath,
                    typeFilter: typeFilter
                })
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error('扫描失败: ' + response.status);
                }
                return response.json();
            })
            .then(data => {
                if (data.error) {
                    showStatus('扫描失败: ' + data.error, 'error');
                    return;
                }
                
                allResults = data.results;
                displayResults(allResults);
                showStatus('扫描完成，共找到 ' + data.count + ' 个键值对', 'success');
            })
            .catch(error => {
                showStatus('扫描失败: ' + error.message, 'error');
            });
        }
        
        // 显示状态信息
        function showStatus(message, type) {
            const statusElement = document.getElementById('status');
            statusElement.innerHTML = message;
            statusElement.className = 'status status-' + type;
            statusElement.style.display = 'block';
        }
        
        // 清空结果
        function clearResults() {
            allResults = [];
            const resultsBody = document.getElementById('resultsBody');
            resultsBody.innerHTML = '<tr><td colspan="4" style="text-align: center; padding: 20px;">请点击"开始扫描"按钮开始扫描 TiKV</td></tr>';
            showStatus('', 'info');
            document.getElementById('status').style.display = 'none';
        }
        
        // 应用筛选
        function applyFilter() {
            const typeFilter = document.getElementById('typeFilter').value;
            if (typeFilter) {
                const filteredResults = allResults.filter(result => result.keyInfo.type === typeFilter);
                displayResults(filteredResults);
            } else {
                displayResults(allResults);
            }
        }
        
        // 显示结果
        function displayResults(results) {
            const resultsBody = document.getElementById('resultsBody');
            
            if (results.length === 0) {
                resultsBody.innerHTML = '<tr><td colspan="4" style="text-align: center; padding: 20px;">没有找到匹配的结果</td></tr>';
                return;
            }
            
            let html = '';
            results.forEach((result, index) => {
                html += '<tr>';
                html += '<td>' + result.keyInfo.type + '</td>';
                html += '<td style="word-break: break-all;">' + result.key + '</td>';
                html += '<td style="word-break: break-all;">' + formatValue(result.valueInfo.value) + '</td>';
                html += '<td>';
                html += '<div class="details">';
                html += '<strong>描述:</strong> ' + result.keyInfo.description + '<br>';
                if (result.keyInfo.subType) {
                    html += '<strong>子类型:</strong> ' + result.keyInfo.subType + '<br>';
                }
                if (result.keyInfo.uuid) {
                    html += '<strong>UUID:</strong> ' + result.keyInfo.uuid + '<br>';
                }
                if (Object.keys(result.keyInfo.details).length > 0) {
                    html += '<strong>Key 详情:</strong><pre>' + JSON.stringify(result.keyInfo.details, null, 2) + '</pre>';
                }
                if (Object.keys(result.valueInfo.details).length > 0) {
                    html += '<strong>Value 详情:</strong><pre>' + JSON.stringify(result.valueInfo.details, null, 2) + '</pre>';
                }
                html += '</div>';
                html += '</td>';
                html += '</tr>';
            });
            
            resultsBody.innerHTML = html;
        }
        
        // 格式化值
        function formatValue(value) {
            if (typeof value === 'string' && value.length > 100) {
                return value.substring(0, 100) + '...';
            }
            return value;
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func main() {
	// 解析命令行参数
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	// 注册路由
	http.HandleFunc("/", frontendHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/scan", scanHandler)

	// 启用 CORS
	handler := enableCORS(http.DefaultServeMux)

	// 启动服务器
	serverAddr := fmt.Sprintf(":%s", *port)
	fmt.Printf("Server started on %s\n", serverAddr)
	if err := http.ListenAndServe(serverAddr, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
