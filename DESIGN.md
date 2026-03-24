# JuiceFS TiKV Key 扫描与解析工具设计文档

## 1. 项目概述

本项目是一个用于扫描和解析 JuiceFS 在 TiKV 数据库中存储的 key 的工具。它能够连接到 TiKV 数据库，扫描所有 key，智能显示 key 的字节表示，并对每个 key 进行详细解析，提取出其中的类型、属性和其他相关信息。

## 2. 系统架构

### 2.1 核心组件

| 组件 | 功能描述 | 文件位置 |
|------|---------|---------|
| KeyInfo 结构体 | 存储解析后的 key 信息 | main.go:12-19 |
| ParseKey 函数 | 解析 JuiceFS TiKV key | main.go:21-56 |
| parseTypeAndContent 函数 | 解析类型前缀和内容 | main.go:58-138 |
| parseAType 函数 | 解析 A 类型（inode 相关） | main.go:140-203 |
| parseCType 函数 | 解析 C 类型（计数器） | main.go:205-236 |
| parseSType 函数 | 解析 S 类型（会话相关） | main.go:238-291 |
| parseDirectString 函数 | 解析直接字符串类型 | main.go:293-327 |
| main 函数 | 主函数，连接 TiKV 并扫描 key | main.go:361-430 |

### 2.2 数据流

1. 连接到 TiKV 数据库
2. 创建只读事务
3. 定义扫描范围（从空字节到最大字节）
4. 创建迭代器并扫描所有 key
5. 对每个 key 进行智能显示和解析
6. 输出解析结果
7. 关闭连接和释放资源

## 3. 核心功能模块

### 3.1 TiKV 连接与扫描

**功能**：连接到指定的 TiKV PD 地址，创建事务和迭代器，扫描所有 key。

**实现**：
- 使用 `txnkv.NewClient` 创建 TiKV 客户端
- 使用 `client.Begin()` 创建只读事务
- 使用 `txn.Iter(startKey, endKey)` 创建迭代器
- 遍历迭代器获取所有 key

**代码位置**：main.go:361-406

### 3.2 Key 智能显示

**功能**：对 key 的每个字节进行判断，可打印字符直接显示，不可打印字符以 16 进制形式显示。

**实现**：
- 遍历 key 的每个字节
- 判断字节是否在可打印范围内（0x20-0x7E）或为制表符、换行符、回车符
- 可打印字符直接转换为字符串，否则以 `\x%02x` 格式显示

**代码位置**：main.go:407-416

### 3.3 Key 解析

**功能**：解析 JuiceFS TiKV key 的结构，提取类型、属性和其他相关信息。

**实现**：
- 识别 key 的 UUID 部分（到 `\xfd` 分隔符为止）
- 解析类型前缀和内容
- 根据不同类型调用相应的解析函数
- 提取并存储解析结果

**代码位置**：main.go:21-56

### 3.4 类型解析

**功能**：根据 key 的类型前缀，调用相应的解析函数进行详细解析。

**实现**：
- 支持 A 类型（inode 相关）
- 支持 C 类型（计数器）
- 支持 S 类型（会话相关），包括 SI、SE、SH、SS 等子类型
- 支持直接字符串类型（如 setting、status、juicefs-fslist-prefix 等）

**代码位置**：main.go:58-138

### 3.5 字节序处理

**功能**：根据不同字段的要求，使用正确的字节序进行解析。

**实现**：
- Inode 编号使用 LittleEndian 解析
- SessionID 和 ChunkIndex 使用 BigEndian 解析
- 其他数值字段根据具体类型选择合适的字节序

**代码位置**：
- LittleEndian: main.go:152, main.go:111
- BigEndian: main.go:85, main.go:93, main.go:100, main.go:108, main.go:178

## 4. 数据结构

### 4.1 KeyInfo 结构体

```go
type KeyInfo struct {
    UUID        string                 // 文件系统 UUID
    Type        string                 // Key 类型
    SubType     string                 // 子类型
    Description string                 // 描述信息
    Details     map[string]interface{} // 详细信息
}
```

**字段说明**：
- `UUID`：文件系统的唯一标识符
- `Type`：Key 的类型（如 A、C、S、SI、SE、Config、FileSystem 等）
- `SubType`：子类型（如 I、D、P、C、S、X 等）
- `Description`：类型的描述信息
- `Details`：详细信息，包含具体的属性值（如 Inode、FileName、SessionID 等）

### 4.2 支持的 Key 类型

| 类型 | 描述 | 子类型 | 详情字段 |
|------|------|--------|----------|
| A | Inode 相关 | I | Inode |
| A | Inode 相关 | D | Inode, FileName |
| A | Inode 相关 | P | Inode |
| A | Inode 相关 | C | Inode, ChunkIndex |
| A | Inode 相关 | S | Inode |
| A | Inode 相关 | X | Inode, AttributeName |
| C | 计数器 | - | - |
| S | 会话相关 | - | - |
| SI | 会话信息 | - | SessionID |
| SE | 会话过期时间 | - | SessionID |
| SH | 会话心跳（遗留） | - | SessionID |
| SS | 持续 inode | - | SessionID, Inode |
| Config | 配置 | - | - |
| FileSystem | 文件系统 | - | FSInfo/FSID |

## 5. 技术实现细节

### 5.1 TiKV 客户端使用

使用 `github.com/tikv/client-go/v2/txnkv` 包创建 TiKV 客户端，连接到指定的 PD 地址。

**代码**：
```go
client, err := txnkv.NewClient([]string{pdAddr})
if err != nil {
    log.Fatalf("❌ Failed to create client: %v", err)
}
defer client.Close()
```

### 5.2 字节序处理

根据 JuiceFS 的存储格式，不同字段使用不同的字节序：
- Inode 编号：LittleEndian
- SessionID 和 ChunkIndex：BigEndian

**代码**：
```go
// LittleEndian 解析 Inode
inode := binary.LittleEndian.Uint64(content[0:8])

// BigEndian 解析 SessionID
sid := binary.BigEndian.Uint64(content[2:10])
```

### 5.3 智能字节显示

对 key 的每个字节进行判断，可打印字符直接显示，不可打印字符以 16 进制形式显示。

**代码**：
```go
keyStr := ""
for _, b := range key {
    if (b >= 0x20 && b <= 0x7E) || b == '\t' || b == '\n' || b == '\r' {
        // 是可打印字符
        keyStr += string(b)
    } else {
        // 不是可打印字符，打印16进制
        keyStr += fmt.Sprintf("\\x%02x", b)
    }
}
```

### 5.4 类型解析

根据 key 的类型前缀，调用相应的解析函数进行详细解析。

**代码**：
```go
switch prefix {
case 'A':
    parseAType(info, content[1:])
case 'C':
    parseCType(info, content[1:])
case 'S':
    // 检查是否是 SI、SE 等独立类型
    if len(content) >= 2 {
        secondChar := content[1]
        switch secondChar {
        case 'E':
            // 解析 SE 类型
        case 'I':
            // 解析 SI 类型
        // ... 其他子类型
        }
    }
// ... 其他类型
}
```

## 6. 使用说明

### 6.1 编译与运行

1. **编译**：
   ```bash
   go build -o tikv-key-scanner main.go
   ```

2. **运行**：
   ```bash
   ./tikv-key-scanner
   ```

### 6.2 配置

- **PD 地址**：默认连接到 `192.168.210.200:2388`，可在代码中修改 `pdAddr` 变量

### 6.3 输出说明

程序会输出以下信息：
1. 连接状态信息
2. 每个 key 的序号和智能显示
3. 每个 key 的详细解析结果，包括：
   - UUID（如果有）
   - 类型
   - 描述
   - 详细信息（如 Inode、FileName、SessionID 等）
4. 扫描完成后的统计信息

## 7. 性能与优化

1. **内存使用**：
   - 每次迭代只处理一个 key，避免一次性加载所有 key 到内存
   - 使用 defer 语句确保资源及时释放

2. **错误处理**：
   - 对关键操作进行错误检查，确保程序稳定运行
   - 详细的错误信息，便于排查问题

3. **扩展性**：
   - 模块化设计，便于添加新的类型解析支持
   - 清晰的代码结构，便于维护和修改

## 8. 未来改进方向

1. **命令行参数支持**：
   - 添加命令行参数，支持指定 PD 地址、证书路径等
   - 支持过滤特定类型的 key

2. **输出格式优化**：
   - 支持 JSON 格式输出，便于后续处理
   - 支持导出解析结果到文件

3. **性能优化**：
   - 支持并发扫描，提高扫描速度
   - 优化内存使用，支持大规模 key 扫描

4. **功能扩展**：
   - 支持解析 value 部分
   - 支持修改和删除 key
   - 支持备份和恢复功能

## 9. 总结

本项目实现了一个功能完整的 JuiceFS TiKV Key 扫描与解析工具，能够：
- 连接到 TiKV 数据库并扫描所有 key
- 智能显示 key 的字节表示
- 详细解析不同类型的 key
- 正确处理字节序问题
- 提供清晰的输出格式

该工具对于了解 JuiceFS 的存储结构、调试问题以及监控系统状态都具有重要的参考价值。