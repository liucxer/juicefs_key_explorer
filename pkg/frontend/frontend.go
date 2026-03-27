package frontend

import (
	"net/http"
)

// FrontendHandler 处理前端页面
func FrontendHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>tikv for juicefs扫描与解析工具</title>
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
            width: 100%;
            margin: 0;
            padding: 20px;
            display: flex;
            flex-direction: column;
            gap: 20px;
        }
        
        .main-content {
            display: flex;
            gap: 20px;
            width: 100%;
        }
        
        .table-section {
            flex: 1;
        }
        
        .detail-panel {
            width: 400px;
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            overflow-y: auto;
            max-height: 80vh;
        }
        
        .detail-panel h3 {
            margin-bottom: 15px;
            color: #2c3e50;
        }
        
        .detail-item {
            margin-bottom: 15px;
            padding-bottom: 10px;
            border-bottom: 1px solid #e0e0e0;
        }
        
        .detail-item:last-child {
            border-bottom: none;
            margin-bottom: 0;
            padding-bottom: 0;
        }
        
        .detail-label {
            font-weight: bold;
            margin-bottom: 5px;
            color: #34495e;
            font-size: 14px;
        }
        
        .detail-value {
            background-color: #f9f9f9;
            padding: 10px;
            border-radius: 4px;
            font-family: monospace;
            white-space: pre-wrap;
            word-break: break-all;
            font-size: 13px;
            line-height: 1.4;
            border-left: 3px solid #2196F3;
        }
        
        .json-key {
            color: #0000ff;
        }
        
        .json-string {
            color: #008000;
        }
        
        .json-number {
            color: #ff0000;
        }
        
        .json-boolean {
            color: #800080;
        }
        
        .json-null {
            color: #808080;
        }
        
        .no-selection {
            text-align: center;
            color: #7f8c8d;
            padding: 40px 20px;
        }
        
        .table-container {
            background: white;
            padding: 10px 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            overflow-x: auto;
            width: 100%;
        }
        
        table {
            width: 100%;
            border-collapse: collapse;
            table-layout: fixed;
        }
        
        tr {
            cursor: pointer;
        }
        
        tr:hover {
            background-color: #f5f5f5;
        }
        
        tr.selected {
            background-color: #e3f2fd;
        }
        
        .tab-navigation {
            display: flex;
            margin-bottom: 20px;
            border-bottom: 1px solid #e0e0e0;
        }
        
        .tab-button {
            background: none;
            border: none;
            padding: 10px 20px;
            cursor: pointer;
            font-size: 14px;
            color: #666;
            border-bottom: 2px solid transparent;
            transition: all 0.3s ease;
        }
        
        .tab-button:hover {
            color: #3498db;
        }
        
        .tab-button.active {
            color: #3498db;
            border-bottom: 2px solid #3498db;
            font-weight: bold;
        }
        
        .connection-config {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        
        .config-row {
            display: flex;
            gap: 15px;
            align-items: flex-end;
            flex-wrap: wrap;
        }
        
        .config-item {
            flex: 1;
            min-width: 200px;
        }
        
        .config-item label {
            display: block;
            margin-bottom: 5px;
            font-size: 14px;
            font-weight: 500;
            color: #555;
        }
        
        .config-item input {
            width: 100%;
            padding: 8px 12px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 14px;
        }
        
        .config-buttons {
            display: flex;
            gap: 10px;
        }
        
        .config-buttons button {
            padding: 8px 16px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 500;
        }
        
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
        }
        
        .btn-settings {
            padding: 8px 16px;
            background-color: #f0f0f0;
            border: 1px solid #ddd;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 500;
        }
        
        .btn-settings:hover {
            background-color: #e0e0e0;
        }
        
        .btn-scan {
            padding: 8px 16px;
            background-color: #2196F3;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            height: 32px;
        }
        
        .btn-scan:hover {
            background-color: #0b7dda;
        }
        
        /* 弹窗样式 */
        .modal {
            display: none;
            position: fixed;
            z-index: 1000;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            overflow: auto;
            background-color: rgba(0,0,0,0.4);
        }
        
        .modal-content {
            background-color: #fefefe;
            margin: 15% auto;
            padding: 20px;
            border: 1px solid #888;
            width: 80%;
            max-width: 800px;
            border-radius: 8px;
            box-shadow: 0 4px 8px rgba(0,0,0,0.2);
        }
        
        .modal-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 1px solid #e0e0e0;
        }
        
        .modal-header h2 {
            margin: 0;
        }
        
        .close-button {
            color: #aaa;
            float: right;
            font-size: 28px;
            font-weight: bold;
            background: none;
            border: none;
            cursor: pointer;
        }
        
        .close-button:hover,
        .close-button:focus {
            color: black;
            text-decoration: none;
            cursor: pointer;
        }
        
        .modal-body {
            margin-bottom: 20px;
        }
        
        .modal-footer {
            display: flex;
            justify-content: flex-end;
            gap: 10px;
            padding-top: 10px;
            border-top: 1px solid #e0e0e0;
        }
        
        /* 新的 Tab 导航样式 */
        .tab-item {
            display: flex;
            align-items: center;
            margin-right: 20px;
        }
        
        .tab-scan-button {
            margin-left: 10px;
            padding: 4px 12px;
            background-color: #3498db;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 12px;
            font-weight: 500;
        }
        
        .tab-scan-button:hover {
            background-color: #2980b9;
        }
        
        h1 {
            margin: 0;
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
            padding: 10px 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            overflow-x: auto;
            width: 100%;
        }
        
        table {
            width: 100%;
            border-collapse: collapse;
            table-layout: fixed;
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
        <div class="header">
            <h1>tikv for juicefs扫描与解析工具</h1>
            <button class="btn-settings" onclick="openSettings()">设置</button>
        </div>
        
        <!-- 设置弹窗 -->
        <div id="settingsModal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h2>连接配置</h2>
                    <button class="close-button" onclick="closeSettings()">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="config-row">
                        <div class="config-item">
                            <label for="pdAddr">PD 地址</label>
                            <input type="text" id="pdAddr" placeholder="例如: 192.168.210.210:2379" value="192.168.210.210:2379">
                        </div>
                        <div class="config-item">
                            <label for="caPath">CA 证书</label>
                            <input type="text" id="caPath" placeholder="/path/to/ca.crt">
                        </div>
                        <div class="config-item">
                            <label for="certPath">客户端证书</label>
                            <input type="text" id="certPath" placeholder="/path/to/client.crt">
                        </div>
                        <div class="config-item">
                            <label for="keyPath">客户端密钥</label>
                            <input type="text" id="keyPath" placeholder="/path/to/client.key">
                        </div>
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn-secondary" onclick="closeSettings()">取消</button>
                    <button class="btn-primary" onclick="closeSettings()">保存</button>
                </div>
            </div>
        </div>
        
        <!-- 主内容区域 -->
        <div class="main-content">
            <!-- 表格区域 -->
            <div class="table-section">
                <div class="table-container">
                    
                    <!-- Tab 导航 -->
                    <div class="tab-navigation">
                        <div class="tab-item">
                            <button class="tab-button" onclick="switchTab('dentry')">Dentry</button>
                        </div>
                        <div class="tab-item">
                            <button class="tab-button active" onclick="switchTab('inode')">inode</button>
                        </div>
                        <div class="tab-item">
                            <button class="tab-button" onclick="switchTab('chunks')">File chunks</button>
                        </div>
                        <div class="tab-item">
                            <button class="tab-button" onclick="switchTab('other')">其他结果</button>
                        </div>
                    </div>
                    
                    <!-- Dentry 筛选条件 -->
                    <div id="dentryFilters" class="dentry-filters" style="display: none; margin-bottom: 20px; padding: 15px; background-color: #f9f9f9; border-radius: 4px;">
                        <h3>筛选条件</h3>
                        <div style="display: flex; gap: 20px; flex-wrap: wrap;">
                            <div style="flex: 1; min-width: 200px;">
                                <label for="dentryUuidFilter">UUID</label>
                                <input type="text" id="dentryUuidFilter" placeholder="输入 UUID" oninput="applyFilter()">
                            </div>
                            <div style="flex: 1; min-width: 200px;">
                                <label for="parentFilter">Parent</label>
                                <input type="text" id="parentFilter" placeholder="输入 parent inode" oninput="applyFilter()">
                            </div>
                            <div style="flex: 1; min-width: 200px;">
                                <label for="nameFilter">Name</label>
                                <input type="text" id="nameFilter" placeholder="输入文件名" oninput="applyFilter()">
                            </div>
                            <div style="flex: 0 0 auto; align-self: flex-end;">
                                <button class="btn-scan" onclick="scanByTab('dentry')">扫描</button>
                            </div>
                        </div>
                    </div>
                    
                    <!-- Inode 筛选条件 -->
                    <div id="inodeFilters" class="inode-filters" style="display: none; margin-bottom: 20px; padding: 15px; background-color: #f9f9f9; border-radius: 4px;">
                        <h3>筛选条件</h3>
                        <div style="display: flex; gap: 20px; flex-wrap: wrap;">
                            <div style="flex: 1; min-width: 200px;">
                                <label for="uuidFilter">UUID</label>
                                <input type="text" id="uuidFilter" placeholder="输入 UUID" oninput="applyFilter()">
                            </div>
                            <div style="flex: 1; min-width: 200px;">
                                <label for="inoFilter">Ino</label>
                                <input type="text" id="inoFilter" placeholder="输入 inode 号" oninput="applyFilter()">
                            </div>
                            <div style="flex: 1; min-width: 200px;">
                                <label for="poolIDFilter">PoolID</label>
                                <input type="text" id="poolIDFilter" placeholder="输入 PoolID" oninput="applyFilter()">
                            </div>
                            <div style="flex: 1; min-width: 200px;">
                                <label for="typeNameFilter">TypeName</label>
                                <input type="text" id="typeNameFilter" placeholder="输入类型名称" oninput="applyFilter()">
                            </div>
                            <div style="flex: 0 0 auto; align-self: flex-end;">
                                <button class="btn-scan" onclick="scanByTab('inode')">扫描</button>
                            </div>
                        </div>
                    </div>
                    
                    <!-- File chunks 筛选条件 -->
                    <div id="chunksFilters" class="chunks-filters" style="display: none; margin-bottom: 20px; padding: 15px; background-color: #f9f9f9; border-radius: 4px;">
                        <h3>筛选条件</h3>
                        <div style="display: flex; gap: 20px; flex-wrap: wrap;">
                            <div style="flex: 1; min-width: 200px;">
                                <label for="chunksUuidFilter">UUID</label>
                                <input type="text" id="chunksUuidFilter" placeholder="输入 UUID" oninput="applyFilter()">
                            </div>
                            <div style="flex: 1; min-width: 200px;">
                                <label for="inodeFilter">Inode</label>
                                <input type="text" id="inodeFilter" placeholder="输入 inode 号" oninput="applyFilter()">
                            </div>
                            <div style="flex: 1; min-width: 200px;">
                                <label for="chunkIndexFilter">ChunkIndex</label>
                                <input type="text" id="chunkIndexFilter" placeholder="输入 ChunkIndex" oninput="applyFilter()">
                            </div>
                            <div style="flex: 0 0 auto; align-self: flex-end;">
                                <button class="btn-scan" onclick="scanByTab('chunks')">扫描</button>
                            </div>
                        </div>
                    </div>
                    
                    <!-- 其他结果筛选条件 -->
                    <div id="otherFilters" class="other-filters" style="display: none; margin-bottom: 20px; padding: 15px; background-color: #f9f9f9; border-radius: 4px;">
                        <h3>筛选条件</h3>
                        <div style="display: flex; gap: 20px; flex-wrap: wrap;">
                            <div style="flex: 1; min-width: 200px;">
                                <label for="otherTypeFilter">类型</label>
                                <input type="text" id="otherTypeFilter" placeholder="输入类型" oninput="applyFilter()">
                            </div>
                            <div style="flex: 1; min-width: 200px;">
                                <label for="otherDescriptionFilter">描述</label>
                                <input type="text" id="otherDescriptionFilter" placeholder="输入描述" oninput="applyFilter()">
                            </div>
                            <div style="flex: 0 0 auto; align-self: flex-end;">
                                <button class="btn-scan" onclick="scanByTab('other')">扫描</button>
                            </div>
                        </div>
                    </div>
                    
                    <table id="resultsTable">
                        <thead>
                            <tr>
                                <th style="width: 8%; min-width: 80px;">类型</th>
                                <th style="width: 15%; min-width: 120px;">描述</th>
                                <th style="width: 15%; min-width: 150px; max-width: 200px;">Key</th>
                                <th style="width: 15%; min-width: 150px; max-width: 200px;">Value</th>
                                <th style="width: 8%; min-width: 80px;">子类型</th>
                                <th style="width: 10%; min-width: 100px;">UUID</th>
                                <th style="width: 12%; min-width: 120px;">Key 详情</th>
                                <th style="width: 12%; min-width: 120px;">Value 详情</th>
                            </tr>
                        </thead>
                        <tbody id="resultsBody">
                            <tr>
                                <td colspan="8" style="text-align: center; padding: 20px;">请点击"开始扫描"按钮开始扫描 TiKV</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
            
            <!-- 详细信息面板 -->
            <div class="detail-panel">
                <h3>详细信息</h3>
                <div id="detailContent" class="no-selection">
                    点击表格行查看详细信息
                </div>
            </div>
        </div>
    </div>
    
    <script>
        let allResults = [];
        let currentTab = 'inode';
        
        // 打开设置弹窗
        function openSettings() {
            document.getElementById('settingsModal').style.display = 'block';
        }
        
        // 关闭设置弹窗
        function closeSettings() {
            document.getElementById('settingsModal').style.display = 'none';
        }
        
        // 按 Tab 类型扫描
        function scanByTab(tab) {
            // 切换到对应的 tab
            switchTab(tab);
            
            // 执行扫描
            scanTiKV();
        }
        
        // 扫描 TiKV
        function scanTiKV() {
            const pdAddr = document.getElementById('pdAddr').value;
            const caPath = document.getElementById('caPath').value;
            const certPath = document.getElementById('certPath').value;
            const keyPath = document.getElementById('keyPath').value;
            
            if (!pdAddr) {
                alert('请输入 PD 地址');
                return;
            }
            
            // 根据当前 tab 确定描述筛选条件
            let descriptionFilter = '';
            if (currentTab === 'inode') {
                descriptionFilter = 'Inode attribute';
            } else if (currentTab === 'dentry') {
                descriptionFilter = 'Dentry';
            } else if (currentTab === 'chunks') {
                descriptionFilter = 'File chunks';
            } else if (currentTab === 'other') {
                descriptionFilter = ''; // 空字符串，表示不进行描述筛选
            }
            
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
                    typeFilter: [], // 空数组，表示不进行类型筛选
                    descriptionFilter: descriptionFilter
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
                    alert('扫描失败: ' + data.error);
                    return;
                }
                
                allResults = data.results;
                applyFilter();
            })
            .catch(error => {
                alert('扫描失败: ' + error.message);
            });
        }
        
        // 切换 Tab
        function switchTab(tab) {
            currentTab = tab;
            
            // 更新 Tab 按钮状态
            document.querySelectorAll('.tab-button').forEach(button => {
                button.classList.remove('active');
            });
            document.querySelector(".tab-button[onclick=\"switchTab('" + tab + "')\"]").classList.add('active');
            
            // 显示对应的筛选条件
            document.getElementById('dentryFilters').style.display = 'none';
            document.getElementById('inodeFilters').style.display = 'none';
            document.getElementById('chunksFilters').style.display = 'none';
            document.getElementById('otherFilters').style.display = 'none';
            
            if (tab === 'dentry') {
                document.getElementById('dentryFilters').style.display = 'block';
            } else if (tab === 'inode') {
                document.getElementById('inodeFilters').style.display = 'block';
            } else if (tab === 'chunks') {
                document.getElementById('chunksFilters').style.display = 'block';
            } else if (tab === 'other') {
                document.getElementById('otherFilters').style.display = 'block';
            }
            
            // 应用筛选
            applyFilter();
        }
        
        // 清空结果
        function clearResults() {
            allResults = [];
            const resultsBody = document.getElementById('resultsBody');
            resultsBody.innerHTML = '<tr><td colspan="4" style="text-align: center; padding: 20px;">请点击"扫描"按钮开始扫描 TiKV</td></tr>';
        }
        
        // 应用筛选
        function applyFilter() {
            let filteredResults = allResults;
            
            // 应用 Tab 过滤
            if (currentTab === 'inode') {
                filteredResults = filteredResults.filter(result => result.keyInfo.description === 'Inode attribute');
                
                // 应用 inode 筛选条件
                const uuidFilter = document.getElementById('uuidFilter');
                const inoFilter = document.getElementById('inoFilter');
                const poolIDFilter = document.getElementById('poolIDFilter');
                const typeNameFilter = document.getElementById('typeNameFilter');
                
                if (uuidFilter) {
                    const uuidValue = uuidFilter.value.trim();
                    if (uuidValue) {
                        filteredResults = filteredResults.filter(result => {
                            const uuid = result.keyInfo.uuid || '';
                            return uuid.toLowerCase().includes(uuidValue.toLowerCase());
                        });
                    }
                }
                
                if (inoFilter) {
                    const inoValue = inoFilter.value.trim();
                    if (inoValue) {
                        filteredResults = filteredResults.filter(result => {
                            const inodeNumber = result.keyInfo.details.Inode || result.keyInfo.details.inode || '';
                            return inodeNumber.toString().includes(inoValue);
                        });
                    }
                }
                
                if (poolIDFilter) {
                    const poolIDValue = poolIDFilter.value.trim();
                    if (poolIDValue) {
                        filteredResults = filteredResults.filter(result => {
                            const poolID = result.valueInfo.details.PoolID || result.valueInfo.details.poolID || '';
                            return poolID.toString().includes(poolIDValue);
                        });
                    }
                }
                
                if (typeNameFilter) {
                    const typeNameValue = typeNameFilter.value.trim();
                    if (typeNameValue) {
                        filteredResults = filteredResults.filter(result => {
                            const typeName = result.valueInfo.details.TypeName || result.valueInfo.details.typeName || '';
                            return typeName.toLowerCase().includes(typeNameValue.toLowerCase());
                        });
                    }
                }
            } else if (currentTab === 'dentry') {
                filteredResults = filteredResults.filter(result => result.keyInfo.description === 'Dentry');
                
                // 应用 dentry 筛选条件
                const dentryUuidFilter = document.getElementById('dentryUuidFilter');
                const parentFilter = document.getElementById('parentFilter');
                const nameFilter = document.getElementById('nameFilter');
                
                if (dentryUuidFilter) {
                    const uuidValue = dentryUuidFilter.value.trim();
                    if (uuidValue) {
                        filteredResults = filteredResults.filter(result => {
                            const uuid = result.keyInfo.uuid || '';
                            return uuid.toLowerCase().includes(uuidValue.toLowerCase());
                        });
                    }
                }
                
                if (parentFilter) {
                    const parentValue = parentFilter.value.trim();
                    if (parentValue) {
                        filteredResults = filteredResults.filter(result => {
                            const parent = result.valueInfo.details.Parent || '';
                            return parent.toString().includes(parentValue);
                        });
                    }
                }
                
                if (nameFilter) {
                    const nameValue = nameFilter.value.trim();
                    if (nameValue) {
                        filteredResults = filteredResults.filter(result => {
                            const fileName = result.keyInfo.details.FileName || result.keyInfo.details.filename || '';
                            return fileName.toLowerCase().includes(nameValue.toLowerCase());
                        });
                    }
                }
            } else if (currentTab === 'chunks') {
                filteredResults = filteredResults.filter(result => result.keyInfo.description === 'File chunks');
                
                // 应用 chunks 筛选条件
                const chunksUuidFilter = document.getElementById('chunksUuidFilter');
                const inodeFilter = document.getElementById('inodeFilter');
                const chunkIndexFilter = document.getElementById('chunkIndexFilter');
                
                if (chunksUuidFilter) {
                    const uuidValue = chunksUuidFilter.value.trim();
                    if (uuidValue) {
                        filteredResults = filteredResults.filter(result => {
                            const uuid = result.keyInfo.uuid || '';
                            return uuid.toLowerCase().includes(uuidValue.toLowerCase());
                        });
                    }
                }
                
                if (inodeFilter) {
                    const inodeValue = inodeFilter.value.trim();
                    if (inodeValue) {
                        filteredResults = filteredResults.filter(result => {
                            const inode = result.keyInfo.details.Inode || result.keyInfo.details.inode || '';
                            return inode.toString().includes(inodeValue);
                        });
                    }
                }
                
                if (chunkIndexFilter) {
                    const chunkIndexValue = chunkIndexFilter.value.trim();
                    if (chunkIndexValue) {
                        filteredResults = filteredResults.filter(result => {
                            const chunkIndex = result.keyInfo.details.ChunkIndex || result.keyInfo.details.chunkIndex || '';
                            return chunkIndex.toString().includes(chunkIndexValue);
                        });
                    }
                }
            } else if (currentTab === 'other') {
                filteredResults = filteredResults.filter(result => {
                    return result.keyInfo.description !== 'Inode attribute' && 
                           result.keyInfo.description !== 'Dentry' && 
                           result.keyInfo.description !== 'File chunks';
                });
                
                // 应用 other 筛选条件
                const otherTypeFilter = document.getElementById('otherTypeFilter');
                const otherDescriptionFilter = document.getElementById('otherDescriptionFilter');
                
                if (otherTypeFilter) {
                    const typeValue = otherTypeFilter.value.trim();
                    if (typeValue) {
                        filteredResults = filteredResults.filter(result => {
                            const type = result.keyInfo.type || '';
                            return type.toLowerCase().includes(typeValue.toLowerCase());
                        });
                    }
                }
                
                if (otherDescriptionFilter) {
                    const descriptionValue = otherDescriptionFilter.value.trim();
                    if (descriptionValue) {
                        filteredResults = filteredResults.filter(result => {
                            const description = result.keyInfo.description || '';
                            return description.toLowerCase().includes(descriptionValue.toLowerCase());
                        });
                    }
                }
            }
            
            // 更新表格
            updateTable(filteredResults);
        }
        
        // 更新表格
        function updateTable(results) {
            const resultsBody = document.getElementById('resultsBody');
            
            if (results.length === 0) {
                resultsBody.innerHTML = '<tr><td colspan="8" style="text-align: center; padding: 20px;">没有找到匹配的结果</td></tr>';
                return;
            }
            
            let html = '';
            for (let i = 0; i < results.length; i++) {
                const result = results[i];
                const keyInfo = result.keyInfo;
                const valueInfo = result.valueInfo;
                
                // 生成表格行
                html += '<tr onclick="showDetails(' + i + ')"><td>' + (keyInfo.type || '-') + '</td><td>' + (keyInfo.description || '-') + '</td><td title="' + result.key + '">' + truncateString(result.key, 30) + '</td><td title="' + (valueInfo.value || '-') + '">' + truncateString((valueInfo.value || '').toString(), 30) + '</td><td>' + (keyInfo.subType || '-') + '</td><td style="min-width: 150px;">' + (keyInfo.uuid || '') + '</td><td title="' + JSON.stringify(keyInfo.details) + '">' + truncateString(JSON.stringify(keyInfo.details), 20) + '</td><td title="' + JSON.stringify(valueInfo.details) + '">' + truncateString(JSON.stringify(valueInfo.details), 20) + '</td></tr>';
            }
            
            resultsBody.innerHTML = html;
        }
        
        // 显示详细信息
        function showDetails(index) {
            const result = allResults[index];
            const keyInfo = result.keyInfo;
            const valueInfo = result.valueInfo;
            
            const detailContent = document.getElementById('detailContent');
            
            // 生成详细信息 HTML
            let html = '';
            
            // Key 详细信息
            html += '<div class="detail-item"><div class="detail-label">Key 详细</div><div class="detail-value">' + formatJSON(keyInfo.details || {}) + '</div></div>';
            
            // Value 详细信息
            html += '<div class="detail-item"><div class="detail-label">Value 详细</div><div class="detail-value">' + formatJSON(valueInfo.details || {}) + '</div></div>';
            
            // 其他信息
            html += '<div class="detail-item"><div class="detail-label">类型</div><div class="detail-value">' + (keyInfo.type || '-') + '</div></div>';
            html += '<div class="detail-item"><div class="detail-label">描述</div><div class="detail-value">' + (keyInfo.description || '-') + '</div></div>';
            html += '<div class="detail-item"><div class="detail-label">子类型</div><div class="detail-value">' + (keyInfo.subType || '-') + '</div></div>';
            html += '<div class="detail-item"><div class="detail-label">UUID</div><div class="detail-value">' + (keyInfo.uuid || '-') + '</div></div>';
            html += '<div class="detail-item"><div class="detail-label">Key</div><div class="detail-value">' + (result.key || '-') + '</div></div>';
            html += '<div class="detail-item"><div class="detail-label">Value</div><div class="detail-value">' + (valueInfo.value || '-') + '</div></div>';
            
            detailContent.innerHTML = html;
        }
        
        // 截断字符串
        function truncateString(str, maxLength) {
            if (str.length <= maxLength) {
                return str;
            }
            return str.substring(0, maxLength) + '...';
        }
        
        // 格式化 JSON 并添加语法高亮
        function formatJSON(obj) {
            const jsonStr = JSON.stringify(obj, null, 2);
            return jsonStr
                .replace(/"([^"]+)"\s*:/g, '<span class="json-key">"$1"</span>:')
                .replace(/:\s*"([^"]+)"/g, ': <span class="json-string">"$1"</span>')
                .replace(/:\s*(\d+)/g, ': <span class="json-number">$1</span>')
                .replace(/:\s*(true|false)/g, ': <span class="json-boolean">$1</span>')
                .replace(/:\s*null/g, ': <span class="json-null">null</span>');
        }
        
        // 页面加载完成后自动加载 inode 数据
        window.onload = function() {
            scanByTab('inode');
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
