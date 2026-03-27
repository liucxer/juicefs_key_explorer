# TiKV for JuiceFS 扫描与解析工具

## 项目简介

这是一个用于扫描和解析 JuiceFS 在 TiKV 中存储结构的工具，帮助用户了解 JuiceFS 如何在 TiKV 中组织和存储数据。

## 功能特性

- 连接到真实的 TiKV 实例进行扫描
- 提供多标签页的用户界面，分别显示不同类型的数据
- 详细展示键值对的信息
- 支持按不同类型进行筛选
- 自动为所有标签页加载数据
- 格式化显示 JSON 数据
- 确保所有数据完整显示，无截断

## 目录结构

```
juicefs_key_explorer/
├── main.go              # 应用入口点
├── go.mod               # Go 模块配置
├── pkg/
│   ├── model/           # 模型定义
│   │   └── model.go
│   ├── parse/           # 解析逻辑
│   │   └── parse.go
│   ├── server/          # 服务器逻辑
│   │   └── server.go
│   └── frontend/        # 前端处理
│       └── frontend.go
```

## 安装与运行

### 前提条件

- Go 1.23 或更高版本
- 访问 TiKV 集群的权限

### 安装步骤

1. 克隆代码库

2. 安装依赖
   ```bash
   go mod tidy
   ```

3. 构建应用
   ```bash
   go build -o juicefs_key_explorer
   ```

4. 运行应用
   ```bash
   ./juicefs_key_explorer
   ```

5. 访问 Web 界面
   打开浏览器，访问 `http://localhost:8080`

## 使用方法

1. 在设置弹窗中配置 TiKV 连接信息
   - PD 地址：TiKV 集群的 PD 地址
   - CA 证书、客户端证书、客户端密钥（如果需要）

2. 点击各个标签页的扫描按钮开始扫描对应类型的数据
   - Dentry：目录项数据
   - inode：inode 属性数据
   - File chunks：文件块数据
   - 其他结果：其他类型的数据

3. 点击表格行查看详细信息

4. 使用筛选条件对数据进行过滤

## 技术实现

- 后端：Go 语言，使用 TiKV client-go 连接 TiKV
- 前端：HTML、CSS、JavaScript
- API：RESTful API

## 注意事项

- 扫描大量数据可能会占用较多资源，请根据实际情况调整
- 确保网络连接稳定，避免扫描过程中断

---

# TiKV for JuiceFS Scanner and Parser

## Project Introduction

This is a tool for scanning and parsing JuiceFS storage structure in TiKV, helping users understand how JuiceFS organizes and stores data in TiKV.

## Features

- Connect to real TiKV instances for scanning
- Provide a multi-tab user interface to display different types of data
- Show detailed information about key-value pairs
- Support filtering by different types
- Auto-load data for all tabs
- Format JSON data for better readability
- Ensure all data is displayed completely without truncation

## Directory Structure

```
juicefs_key_explorer/
├── main.go              # Application entry point
├── go.mod               # Go module configuration
├── pkg/
│   ├── model/           # Model definitions
│   │   └── model.go
│   ├── parse/           # Parsing logic
│   │   └── parse.go
│   ├── server/          # Server logic
│   │   └── server.go
│   └── frontend/        # Frontend processing
│       └── frontend.go
```

## Installation and Running

### Prerequisites

- Go 1.23 or higher
- Permission to access the TiKV cluster

### Installation Steps

1. Clone the repository

2. Install dependencies
   ```bash
   go mod tidy
   ```

3. Build the application
   ```bash
   go build -o juicefs_key_explorer
   ```

4. Run the application
   ```bash
   ./juicefs_key_explorer
   ```

5. Access the web interface
   Open a browser and visit `http://localhost:8080`

## Usage

1. Configure TiKV connection information in the settings modal
   - PD Address: PD address of the TiKV cluster
   - CA Certificate, Client Certificate, Client Key (if needed)

2. Click the scan button on each tab to start scanning corresponding type of data
   - Dentry: Directory entry data
   - inode: Inode attribute data
   - File chunks: File chunk data
   - Other results: Other types of data

3. Click on table rows to view detailed information

4. Use filter conditions to filter data

## Technical Implementation

- Backend: Go language, using TiKV client-go to connect to TiKV
- Frontend: HTML, CSS, JavaScript
- API: RESTful API

## Notes

- Scanning large amounts of data may consume more resources, please adjust according to actual情况
- Ensure network connection is stable to avoid interruption during scanning
