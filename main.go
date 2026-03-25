package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tikv/client-go/v2/config"
	"github.com/tikv/client-go/v2/txnkv"
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
	PdAddr            string   `json:"pdAddr"`
	CaPath            string   `json:"caPath"`
	CertPath          string   `json:"certPath"`
	KeyPath           string   `json:"keyPath"`
	TypeFilter        []string `json:"typeFilter"`
	DescriptionFilter string   `json:"descriptionFilter"`
}

// ParseKey 解析 JuiceFS TiKV key
func ParseKey(key []byte) *KeyInfo {
	info := &KeyInfo{
		Details:      make(map[string]interface{}),
		ValueDetails: make(map[string]interface{}),
	}

	keyStr := string(key)
	// 特殊处理 juicefs-fslist-prefix 类型
	if strings.HasPrefix(keyStr, "juicefs-fslist-prefix") {
		info.UUID = ""
		parseDirectString(info, key)
		return info
	}

	// 查找 UUID 和类型前缀的分隔符 \xfd
	uuidEnd := -1
	for i, b := range key {
		if b == 0xfd {
			uuidEnd = i
			break
		}
	}

	// 提取 UUID
	if uuidEnd > 0 {
		info.UUID = string(key[:uuidEnd])
		// 解析类型前缀和内容
		parseTypeAndContent(info, key[uuidEnd+1:])
	} else {
		// 直接字符串类型
		info.UUID = ""
		parseDirectString(info, key)
	}

	return info
}

// parseTypeAndContent 解析类型前缀和内容
func parseTypeAndContent(info *KeyInfo, content []byte) {
	if len(content) == 0 {
		info.Type = "Unknown"
		info.Description = "Empty content"
		return
	}

	// 第一个字节是类型前缀
	prefix := content[0]

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
				info.Type = "SE"
				info.Description = "Session expire time"
				info.SubType = ""
				if len(content) >= 10 {
					sid := binary.BigEndian.Uint64(content[2:10])
					info.Details["SessionID"] = sid
				}
			case 'I':
				info.Type = "SI"
				info.Description = "Session info"
				info.SubType = ""
				if len(content) >= 10 {
					sid := binary.BigEndian.Uint64(content[2:10])
					info.Details["SessionID"] = sid
				}
			case 'H':
				info.Type = "SH"
				info.Description = "Session heartbeat (legacy)"
				info.SubType = ""
				if len(content) >= 10 {
					sid := binary.BigEndian.Uint64(content[2:10])
					info.Details["SessionID"] = sid
				}
			case 'S':
				info.Type = "SS"
				info.Description = "Sustained inode"
				info.SubType = ""
				if len(content) >= 10 {
					sid := binary.BigEndian.Uint64(content[2:10])
					info.Details["SessionID"] = sid
					if len(content) >= 18 {
						inode := binary.LittleEndian.Uint64(content[10:18])
						info.Details["Inode"] = inode
					}
				}
			default:
				// 其他 S 类型
				info.Type = "S"
				info.Description = "Session related"
				info.SubType = ""
				parseDirectString(info, append([]byte{0xfd}, content...))
			}
		}
	case 's':
		// 处理 setting 和 status 等直接字符串
		subType := string(content)
		if subType == "setting" || subType == "status" {
			parseDirectString(info, content)
		} else {
			info.Type = fmt.Sprintf("Unknown (0x%02x)", prefix)
			info.Description = "Unknown type prefix"
			info.Details["RawContent"] = fmt.Sprintf("0x%x", content)
		}
	default:
		info.Type = fmt.Sprintf("Unknown (0x%02x)", prefix)
		info.Description = "Unknown type prefix"
		info.Details["RawContent"] = fmt.Sprintf("0x%x", content)
	}
}

// parseAType 解析 A 类型（inode 相关）
func parseAType(info *KeyInfo, content []byte) {
	info.Type = "A"
	info.Description = "Inode related"
	info.SubType = ""

	if len(content) == 0 {
		return
	}

	// 解析 inode 编号（前8字节）
	if len(content) >= 8 {
		inode := binary.LittleEndian.Uint64(content[0:8])
		info.Details["Inode"] = inode

		// 检查剩余部分是否有子类型
		if len(content) > 8 {
			subType := content[8]
			switch subType {
			case 'I':
				info.SubType = "I"
				info.Description = "Inode attribute"
			case 'D':
				info.SubType = "D"
				info.Description = "Dentry"
				if len(content) > 9 {
					// 解析文件名
					fileName := string(content[9:])
					info.Details["FileName"] = fileName
				}
			case 'P':
				info.SubType = "P"
				info.Description = "Parents (for hard links)"
			case 'C':
				info.SubType = "C"
				info.Description = "File chunks"
				if len(content) > 12 {
					indx := binary.BigEndian.Uint32(content[9:13])
					info.Details["ChunkIndex"] = indx
				}
			case 'S':
				info.SubType = "S"
				info.Description = "Symlink target"
			case 'X':
				info.SubType = "X"
				info.Description = "Extended attribute"
				if len(content) > 9 {
					attrName := string(content[9:])
					info.Details["AttributeName"] = attrName
				}
			default:
				// 对于其他类型，不设置子类型
				info.Details["RawContent"] = fmt.Sprintf("0x%x", content[8:])
			}
		} else {
			info.Description = "Inode related"
			info.Details["RawContent"] = fmt.Sprintf("0x%x", content[8:])
		}
	} else {
		// 内容长度不足8字节，无法解析inode
		info.Description = "Inode related"
		info.Details["RawContent"] = fmt.Sprintf("0x%x", content)
	}
}

// parseCType 解析 C 类型（计数器）
func parseCType(info *KeyInfo, content []byte) {
	info.Type = "C"
	info.Description = "Counter"
	info.SubType = ""

	counterName := string(content)

	// 常见计数器名称
	switch counterName {
	case "lastCleanupFiles":
		info.Description = "Last cleanup files timestamp"
	case "lastCleanupSessions":
		info.Description = "Last cleanup sessions timestamp"
	case "nextChunk":
		info.Description = "Next chunk ID"
	case "nextCleanupSlices":
		info.Description = "Next cleanup slices timestamp"
	case "nextInode":
		info.Description = "Next inode number"
	case "nextSession":
		info.Description = "Next session ID"
	case "nextTrash":
		info.Description = "Next trash ID"
	case "totalInodes":
		info.Description = "Total inodes count"
	case "usedSpace":
		info.Description = "Used space"
	default:
		info.Description = "Unknown counter"
	}
}

// parseSType 解析 S 类型（会话相关）
func parseSType(info *KeyInfo, content []byte) {
	info.Type = "S"
	info.Description = "Session related"

	if len(content) == 0 {
		info.SubType = "Unknown"
		return
	}

	// 检查是否是 SE 或 SI 等子类型
	if len(content) >= 1 {
		secondChar := content[0]
		switch secondChar {
		case 'E':
			info.SubType = "SE"
			info.Description = "Session expire time"
			if len(content) > 1 {
				sid := binary.BigEndian.Uint64(content[1:9])
				info.Details["SessionID"] = sid
			}
		case 'I':
			info.SubType = "SI"
			info.Description = "Session info"
			if len(content) > 1 {
				sid := binary.BigEndian.Uint64(content[1:9])
				info.Details["SessionID"] = sid
			}
		case 'H':
			info.SubType = "SH"
			info.Description = "Session heartbeat (legacy)"
			if len(content) > 1 {
				sid := binary.BigEndian.Uint64(content[1:9])
				info.Details["SessionID"] = sid
			}
		case 'S':
			info.SubType = "SS"
			info.Description = "Sustained inode"
			if len(content) > 1 {
				sid := binary.BigEndian.Uint64(content[1:9])
				info.Details["SessionID"] = sid
				if len(content) > 9 {
					inode := binary.LittleEndian.Uint64(content[9:17])
					info.Details["Inode"] = inode
				}
			}
		default:
			// 可能是直接字符串类型
			parseDirectString(info, append([]byte{0xfd}, content...))
		}
	} else {
		info.SubType = "Unknown"
	}
}

// parseDirectString 解析直接字符串类型
func parseDirectString(info *KeyInfo, content []byte) {
	contentStr := string(content)
	info.SubType = ""

	if strings.HasPrefix(contentStr, "juicefs-fslist-prefix") {
		info.Type = "FileSystem"
		info.Description = "JuiceFS file system list"
		if len(contentStr) > len("juicefs-fslist-prefix") {
			// 找到 \xfd 分隔符的位置
			fsInfoPart := contentStr[len("juicefs-fslist-prefix"):]
			// 跳过 \xfd 和 FSIF 前缀，只提取 UUID 部分
			if len(fsInfoPart) > 5 { // 5 = len("\xfdFSIF")
				uuidPart := fsInfoPart[5:]
				info.Details["FSInfo"] = uuidPart
			} else {
				info.Details["FSInfo"] = fsInfoPart
			}
		}
	} else if contentStr == "setting" {
		info.Type = "Config"
		info.Description = "JuiceFS configuration"
	} else if contentStr == "status" {
		info.Type = "Config"
		info.Description = "JuiceFS status"
	} else if strings.HasPrefix(contentStr, "FSIF") {
		info.Type = "FileSystem"
		info.Description = "JuiceFS file system info"
		info.Details["FSID"] = contentStr[len("FSIF"):]
	} else {
		info.Type = "Unknown"
		info.Description = "Unknown direct string"
		info.Details["RawContent"] = contentStr
	}
}

// ParseValue 解析 JuiceFS TiKV value
func ParseValue(keyInfo *KeyInfo, value []byte) *ValueInfo {
	info := &ValueInfo{
		Details: make(map[string]interface{}),
	}

	switch keyInfo.Type {
	case "Config":
		if keyInfo.SubType == "setting" || keyInfo.Description == "JuiceFS configuration" {
			parseSettingValue(info, value)
		}
	case "C":
		parseCounterValue(info, value)
	case "SE":
		parseSessionValue(info, value)
	case "SI":
		parseSessionInfoValue(info, value)
	case "A":
		if keyInfo.SubType == "I" {
			parseNodeValue(info, value)
		} else if keyInfo.SubType == "D" {
			parseEdgeValue(info, value)
		} else if keyInfo.SubType == "C" {
			parseChunkValue(info, value)
		} else if keyInfo.SubType == "S" {
			parseSymlinkValue(info, value)
		} else if keyInfo.SubType == "X" {
			parseXattrValue(info, value)
		}
	default:
		// 对于未知类型，直接显示原始值
		info.Type = "Unknown"
		info.Description = "Unknown value type"
		info.Value = fmt.Sprintf("0x%x", value)
	}

	return info
}

// parseSettingValue 解析 Setting 类型 value（JSON 格式）
func parseSettingValue(info *ValueInfo, value []byte) {
	info.Type = "Setting"
	info.Description = "JSON format filesystem configuration"
	info.Value = string(value)

	// 尝试解析 JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(value, &jsonData); err == nil {
		info.Details = jsonData
	} else {
		info.Details["RawValue"] = string(value)
		info.Details["Error"] = fmt.Sprintf("Failed to parse JSON: %v", err)
	}
}

// parseCounterValue 解析 Counter 类型 value（整数）
func parseCounterValue(info *ValueInfo, value []byte) {
	info.Type = "Counter"
	info.Description = "Integer value"

	// 尝试解析为整数
	if len(value) == 8 {
		// 64-bit integer (big-endian for most counters)
		val := binary.BigEndian.Uint64(value)
		info.Value = val
		info.Details["Value"] = val
	} else {
		// 尝试解析为字符串
		valStr := string(value)
		info.Value = valStr
		info.Details["RawValue"] = valStr
	}
}

// parseSessionValue 解析 Session 类型 value（timestamp）
func parseSessionValue(info *ValueInfo, value []byte) {
	info.Type = "Session"
	info.Description = "Session expiration timestamp"

	if len(value) == 8 {
		// 64-bit timestamp (big-endian)
		timestamp := binary.BigEndian.Uint64(value)
		info.Value = timestamp
		info.Details["Timestamp"] = timestamp
		info.Details["Time"] = time.Unix(int64(timestamp), 0).Format(time.RFC3339)
	} else {
		info.Value = fmt.Sprintf("0x%x", value)
		info.Details["RawValue"] = fmt.Sprintf("0x%x", value)
	}
}

// parseSessionInfoValue 解析 SessionInfo 类型 value（JSON 格式）
func parseSessionInfoValue(info *ValueInfo, value []byte) {
	info.Type = "SessionInfo"
	info.Description = "JSON format session information"
	info.Value = string(value)

	// 尝试解析 JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(value, &jsonData); err == nil {
		info.Details = jsonData
	} else {
		info.Details["RawValue"] = string(value)
		info.Details["Error"] = fmt.Sprintf("Failed to parse JSON: %v", err)
	}
}

// BufferReader 用于按顺序读取二进制数据
type BufferReader struct {
	buf []byte
	pos int
}

// NewBufferReader 创建一个新的 BufferReader
func NewBufferReader(buf []byte) *BufferReader {
	return &BufferReader{
		buf: buf,
		pos: 0,
	}
}

// Get8 读取一个字节
func (r *BufferReader) Get8() uint8 {
	if r.pos >= len(r.buf) {
		return 0
	}
	val := r.buf[r.pos]
	r.pos++
	return val
}

// Get16 读取两个字节（大端序）
func (r *BufferReader) Get16() uint16 {
	if r.pos+2 > len(r.buf) {
		return 0
	}
	val := binary.BigEndian.Uint16(r.buf[r.pos : r.pos+2])
	r.pos += 2
	return val
}

// Get32 读取四个字节（大端序）
func (r *BufferReader) Get32() uint32 {
	if r.pos+4 > len(r.buf) {
		return 0
	}
	val := binary.BigEndian.Uint32(r.buf[r.pos : r.pos+4])
	r.pos += 4
	return val
}

// Get64 读取八个字节（大端序）
func (r *BufferReader) Get64() uint64 {
	if r.pos+8 > len(r.buf) {
		return 0
	}
	val := binary.BigEndian.Uint64(r.buf[r.pos : r.pos+8])
	r.pos += 8
	return val
}

// Left 返回剩余的字节数
func (r *BufferReader) Left() int {
	return len(r.buf) - r.pos
}

// 定义文件系统类型常量
const (
	TypeFile      = 1 // type for regular file
	TypeDirectory = 2 // type for directory
	TypeSymlink   = 3 // type for symlink
	TypeFIFO      = 4 // type for FIFO node
	TypeBlockDev  = 5 // type for block device
	TypeCharDev   = 6 // type for character device
	TypeSocket    = 7 // type for socket
)

// getTypeName 根据类型值获取类型名称
func getTypeName(t uint8) string {
	switch t {
	case TypeFile:
		return "File"
	case TypeDirectory:
		return "Directory"
	case TypeSymlink:
		return "Symlink"
	case TypeFIFO:
		return "FIFO"
	case TypeBlockDev:
		return "BlockDev"
	case TypeCharDev:
		return "CharDev"
	case TypeSocket:
		return "Socket"
	default:
		return "Unknown"
	}
}

// parseNodeValue 解析 Node 类型 value（二进制 Attr）
func parseNodeValue(info *ValueInfo, value []byte) {
	info.Type = "Node"
	info.Description = "Binary encoded file attributes"
	info.Value = fmt.Sprintf("0x%x", value)

	// 尝试解析 Attr 结构
	rb := NewBufferReader(value)

	// 按照 baseMeta.parseAttr 函数的顺序解析字段
	flags := rb.Get8()
	mode := rb.Get16()
	typ := uint8(mode >> 12) // 从 Mode 中提取类型
	mode &= 0xfff            // 保留 Mode 的低 12 位
	uid := rb.Get32()
	gid := rb.Get32()
	atime := int64(rb.Get64())
	atimensec := rb.Get32()
	mtime := int64(rb.Get64())
	mtimensec := rb.Get32()
	ctime := int64(rb.Get64())
	ctimensec := rb.Get32()
	nlink := rb.Get32()
	length := rb.Get64()
	rdev := rb.Get32()

	// 存储解析结果
	info.Details["Flags"] = flags
	info.Details["Mode"] = mode
	info.Details["Uid"] = uid
	info.Details["Gid"] = gid
	info.Details["Atime"] = atime
	info.Details["Atimensec"] = atimensec
	info.Details["Mtime"] = mtime
	info.Details["Mtimensec"] = mtimensec
	info.Details["Ctime"] = ctime
	info.Details["Ctimensec"] = ctimensec
	info.Details["Nlink"] = nlink
	info.Details["Length"] = length
	info.Details["Rdev"] = rdev

	// 解析可选字段
	if rb.Left() >= 8 {
		parent := rb.Get64()
		info.Details["Parent"] = parent
	}

	if rb.Left() >= 8 {
		poolID := rb.Get64()
		info.Details["PoolID"] = poolID
	}

	if rb.Left() >= 1 {
		hotStat := rb.Get8()
		info.Details["HotStat"] = hotStat
	}

	info.Details["Full"] = true

	// 添加人类可读的时间格式
	info.Details["AtimeStr"] = time.Unix(atime, int64(atimensec*1000)).Format(time.RFC3339)
	info.Details["MtimeStr"] = time.Unix(mtime, int64(mtimensec*1000)).Format(time.RFC3339)
	info.Details["CtimeStr"] = time.Unix(ctime, int64(ctimensec*1000)).Format(time.RFC3339)

	// 添加文件类型的可读名称
	info.Details["TypeName"] = getTypeName(typ)
}

// parseEdgeValue 解析 Edge 类型 value（二进制 {type, inode}）
func parseEdgeValue(info *ValueInfo, value []byte) {
	info.Type = "Edge"
	info.Description = "Binary encoded {type, inode}"
	info.Value = fmt.Sprintf("0x%x", value)

	if len(value) >= 9 {
		typ := value[0]
		info.Details["Type"] = getTypeName(typ)
		info.Details["Inode"] = binary.BigEndian.Uint64(value[1:9])
	} else {
		info.Details["RawValue"] = fmt.Sprintf("0x%x", value)
	}
}

// parseChunkValue 解析 Chunk 类型 value（二进制 Slices）
func parseChunkValue(info *ValueInfo, value []byte) {
	info.Type = "Chunk"
	info.Description = "Binary encoded slices"
	info.Value = fmt.Sprintf("0x%x", value)

	// 每个 Slice 占 24 字节
	sliceSize := 24
	if len(value) > 0 {
		slices := make([]map[string]interface{}, 0)
		for i := 0; i < len(value); i += sliceSize {
			if i+sliceSize <= len(value) {
				sliceData := value[i : i+sliceSize]
				sliceInfo := map[string]interface{}{
					"Pos":  binary.LittleEndian.Uint32(sliceData[0:4]),
					"ID":   binary.LittleEndian.Uint64(sliceData[4:12]),
					"Size": binary.LittleEndian.Uint32(sliceData[12:16]),
					"Off":  binary.LittleEndian.Uint32(sliceData[16:20]),
					"Len":  binary.LittleEndian.Uint32(sliceData[20:24]),
				}
				slices = append(slices, sliceInfo)
			}
		}
		info.Details["Slices"] = slices
		info.Details["Count"] = len(slices)
	} else {
		info.Details["RawValue"] = fmt.Sprintf("0x%x", value)
	}
}

// parseSymlinkValue 解析 Symlink 类型 value（target）
func parseSymlinkValue(info *ValueInfo, value []byte) {
	info.Type = "Symlink"
	info.Description = "Symlink target path"
	info.Value = string(value)
	info.Details["Target"] = string(value)
}

// parseXattrValue 解析 Xattr 类型 value（xattr value）
func parseXattrValue(info *ValueInfo, value []byte) {
	info.Type = "Xattr"
	info.Description = "Extended attribute value"
	info.Value = fmt.Sprintf("0x%x", value)
	info.Details["RawValue"] = string(value)
}

// 生成智能显示的 key 字符串
func formatKey(key []byte) string {
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
	return keyStr
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

	// 验证 PD 地址
	if req.PdAddr == "" {
		http.Error(w, "PD address is required", http.StatusBadRequest)
		return
	}

	// 配置 TiKV 客户端
	if req.CaPath != "" {
		config.UpdateGlobal(func(conf *config.Config) {
			conf.Security = config.NewSecurity(
				req.CaPath,
				req.CertPath,
				req.KeyPath,
				[]string{})
		})
	}

	// 创建 TiKV 客户端
	client, err := txnkv.NewClient([]string{req.PdAddr})
	if err != nil {
		http.Error(w, "Failed to create TiKV client: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	// 创建只读事务
	txn, err := client.Begin()
	if err != nil {
		http.Error(w, "Failed to begin transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer txn.Rollback()

	// 定义扫描范围
	startKey := []byte{}
	endKey := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	// 创建迭代器
	iter, err := txn.Iter(startKey, endKey)
	if err != nil {
		http.Error(w, "Failed to create iterator: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer iter.Close()

	// 扫描并解析键值对
	var results []ScanResult
	count := 0

	for iter.Valid() {
		count++
		key := iter.Key()
		value := iter.Value()

		// 解析 key
		keyInfo := ParseKey(key)

		// 应用类型筛选
		if len(req.TypeFilter) > 0 {
			match := false
			for _, filterType := range req.TypeFilter {
				if keyInfo.Type == filterType {
					match = true
					break
				}
			}
			if !match {
				iter.Next()
				continue
			}
		}

		// 应用描述筛选
		if req.DescriptionFilter != "" {
			if keyInfo.Description != req.DescriptionFilter {
				iter.Next()
				continue
			}
		}

		// 解析 value
		valueInfo := ParseValue(keyInfo, value)

		// 生成智能显示的 key 字符串
		formattedKey := formatKey(key)

		// 添加到结果列表
		results = append(results, ScanResult{
			Key:       formattedKey,
			KeyInfo:   keyInfo,
			ValueInfo: valueInfo,
		})

		iter.Next()
	}

	// 返回结果
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
                            const parent = result.keyInfo.details.Inode || result.keyInfo.details.inode || '';
                            return parent.toString().includes(parentValue);
                        });
                    }
                }
                
                if (nameFilter) {
                    const nameValue = nameFilter.value.trim();
                    if (nameValue) {
                        filteredResults = filteredResults.filter(result => {
                            const name = result.keyInfo.details.FileName || result.keyInfo.details.fileName || result.keyInfo.details.Name || result.keyInfo.details.name || '';
                            return name.toLowerCase().includes(nameValue.toLowerCase());
                        });
                    }
                }
            } else if (currentTab === 'chunks') {
                filteredResults = filteredResults.filter(result => result.keyInfo.description === 'File chunks');
            } else if (currentTab === 'other') {
                filteredResults = filteredResults.filter(result => 
                    result.keyInfo.description !== 'Inode attribute' &&
                    result.keyInfo.description !== 'Dentry' &&
                    result.keyInfo.description !== 'File chunks'
                );
            }
            
            displayResults(filteredResults);
        }
        
        // 显示结果
        function displayResults(results) {
            const resultsBody = document.getElementById('resultsBody');
            const resultsTable = document.getElementById('resultsTable');
            const tableHead = resultsTable.querySelector('thead tr');
            
            // 根据当前 tab 调整表头
            if (currentTab === 'inode') {
                tableHead.innerHTML = 
                    '<th style="width: 30%; min-width: 350px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">UUID</th>' +
                    '<th style="width: 10%; min-width: 80px;">ino 号</th>' +
                    '<th style="width: 8%; min-width: 60px;">PoolID</th>' +
                    '<th style="width: 8%; min-width: 60px;">Length</th>' +
                    '<th style="width: 8%; min-width: 60px;">Mode</th>' +
                    '<th style="width: 15%; min-width: 120px;">TypeName</th>' +
                    '<th style="width: 21%; min-width: 150px;">MtimeStr</th>';
            } else if (currentTab === 'dentry') {
                tableHead.innerHTML = 
                    '<th style="width: 30%; min-width: 350px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">UUID</th>' +
                    '<th style="width: 10%; min-width: 100px;">parent</th>' +
                    '<th style="width: 20%; min-width: 150px;">name</th>' +
                    '<th style="width: 15%; min-width: 100px;">type</th>' +
                    '<th style="width: 15%; min-width: 100px;">ino</th>';
            } else if (currentTab === 'chunks') {
                tableHead.innerHTML = 
                    '<th style="width: 30%; min-width: 350px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">UUID</th>' +
                    '<th style="width: 15%; min-width: 100px;">Inode</th>' +
                    '<th style="width: 15%; min-width: 100px;">ChunkIndex</th>' +
                    '<th style="width: 40%; min-width: 150px;">Value</th>';
            } else {
                tableHead.innerHTML = 
                    '<th style="width: 30%; min-width: 350px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">UUID</th>' +
                    '<th style="width: 20%; min-width: 150px;">描述</th>' +
                    '<th style="width: 25%; min-width: 200px;">Key 详情</th>' +
                    '<th style="width: 25%; min-width: 200px;">Value 详情</th>';
            }
            
            if (results.length === 0) {
                let colSpan = 4; // 默认列数
                if (currentTab === 'inode') {
                    colSpan = 7;
                } else if (currentTab === 'dentry') {
                    colSpan = 5;
                } else if (currentTab === 'chunks') {
                    colSpan = 4;
                }
                resultsBody.innerHTML = '<tr><td colspan="' + colSpan + '" style="text-align: center; padding: 20px;">请点击"扫描"按钮开始扫描 TiKV</td></tr>';
                return;
            }
            
            let html = '';
            results.forEach((result, index) => {
                html += '<tr data-index="' + index + '" onclick="showDetails(' + index + ')">';
                
                if (currentTab === 'inode') {
                    // Inode attribute tab 显示七列：UUID、ino 号、PoolID、Length、Mode、TypeName、MtimeStr
                    const inodeNumber = result.keyInfo.details.Inode || result.keyInfo.details.inode || '-';
                    const poolID = result.valueInfo.details.poolID !== undefined ? result.valueInfo.details.poolID : (result.valueInfo.details.PoolID !== undefined ? result.valueInfo.details.PoolID : '-');
                    const length = result.valueInfo.details.Length || result.valueInfo.details.length || '-';
                    const mode = result.valueInfo.details.Mode || result.valueInfo.details.mode || '-';
                    const typeName = result.valueInfo.details.TypeName || result.valueInfo.details.typeName || '-';
                    const mtimeStr = result.valueInfo.details.MtimeStr || result.valueInfo.details.mtimeStr || '-';
                    
                    html += '<td style="width: 30%; min-width: 350px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + (result.keyInfo.uuid || '') + '">' + (result.keyInfo.uuid || '-') + '</td>';
                    html += '<td style="width: 10%; min-width: 80px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + inodeNumber + '</td>';
                    html += '<td style="width: 8%; min-width: 60px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + poolID + '</td>';
                    html += '<td style="width: 8%; min-width: 60px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + length + '</td>';
                    html += '<td style="width: 8%; min-width: 60px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + mode + '</td>';
                    html += '<td style="width: 15%; min-width: 120px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + typeName + '</td>';
                    html += '<td style="width: 21%; min-width: 150px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + mtimeStr + '</td>';
                } else if (currentTab === 'chunks') {
                    // File chunks tab 显示四列：UUID、Inode、ChunkIndex、Value
                    const inode = result.keyInfo.details.Inode || result.keyInfo.details.inode || '-';
                    const chunkIndex = result.keyInfo.details.ChunkIndex !== undefined ? result.keyInfo.details.ChunkIndex : (result.keyInfo.details.chunkindex !== undefined ? result.keyInfo.details.chunkindex : '-');
                    
                    html += '<td style="width: 30%; min-width: 350px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + (result.keyInfo.uuid || '') + '">' + (result.keyInfo.uuid || '-') + '</td>';
                    html += '<td style="width: 15%; min-width: 100px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + inode + '</td>';
                    html += '<td style="width: 15%; min-width: 100px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + chunkIndex + '</td>';
                    html += '<td style="width: 40%; min-width: 150px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + formatValue(result.valueInfo.value) + '">' + truncateText(formatValue(result.valueInfo.value), 50) + '</td>';

                } else if (currentTab === 'dentry') {
                    // Dentry tab 只显示五列：UUID、parent、name、type、ino
                    const parent = result.keyInfo.details.Inode || result.keyInfo.details.inode || '-';
                    const name = result.keyInfo.details.FileName || result.keyInfo.details.fileName || result.keyInfo.details.Name || result.keyInfo.details.name || '-';
                    const type = result.valueInfo.details.Type || result.valueInfo.details.type || '-';
                    const ino = result.valueInfo.details.Inode || result.valueInfo.details.inode || '-';
                    html += '<td style="width: 30%; min-width: 350px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + (result.keyInfo.uuid || '') + '">' + (result.keyInfo.uuid || '-') + '</td>';
                    html += '<td style="width: 10%; min-width: 100px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + parent + '</td>';
                    html += '<td style="width: 20%; min-width: 150px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + name + '</td>';
                    html += '<td style="width: 15%; min-width: 100px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + type + '</td>';
                    html += '<td style="width: 15%; min-width: 100px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + ino + '</td>';
                } else {
                    // 其他 tab 显示四列：UUID、描述、Key 详情、Value 详情
                    html += '<td style="width: 30%; min-width: 350px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + (result.keyInfo.uuid || '') + '">' + (result.keyInfo.uuid || '-') + '</td>';
                    html += '<td style="width: 20%; min-width: 150px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + (result.keyInfo.description || '-') + '</td>';
                    html += '<td style="width: 25%; min-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + (Object.keys(result.keyInfo.details).length > 0 ? JSON.stringify(result.keyInfo.details, null, 2) : '-') + '">' + (Object.keys(result.keyInfo.details).length > 0 ? formatDetails(result.keyInfo.details) : '-') + '</td>';
                    html += '<td style="width: 25%; min-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + (Object.keys(result.valueInfo.details).length > 0 ? JSON.stringify(result.valueInfo.details, null, 2) : '-') + '">' + (Object.keys(result.valueInfo.details).length > 0 ? formatDetails(result.valueInfo.details) : '-') + '</td>';
                }
                
                html += '</tr>';
            });
            
            resultsBody.innerHTML = html;
        }
        
        // 切换 Tab
        function switchTab(tab) {
            currentTab = tab;
            
            // 更新 Tab 按钮状态
            document.querySelectorAll('.tab-button').forEach(button => {
                button.classList.remove('active');
            });
            document.querySelector('.tab-button[onclick="switchTab(\'' + tab + '\')"]').classList.add('active');
            
            // 显示或隐藏筛选条件
            const dentryFilters = document.getElementById('dentryFilters');
            const inodeFilters = document.getElementById('inodeFilters');
            const chunksFilters = document.getElementById('chunksFilters');
            const otherFilters = document.getElementById('otherFilters');
            
            if (tab === 'dentry') {
                dentryFilters.style.display = 'block';
                inodeFilters.style.display = 'none';
                chunksFilters.style.display = 'none';
                otherFilters.style.display = 'none';
            } else if (tab === 'inode') {
                dentryFilters.style.display = 'none';
                inodeFilters.style.display = 'block';
                chunksFilters.style.display = 'none';
                otherFilters.style.display = 'none';
            } else if (tab === 'chunks') {
                dentryFilters.style.display = 'none';
                inodeFilters.style.display = 'none';
                chunksFilters.style.display = 'block';
                otherFilters.style.display = 'none';
            } else if (tab === 'other') {
                dentryFilters.style.display = 'none';
                inodeFilters.style.display = 'none';
                chunksFilters.style.display = 'none';
                otherFilters.style.display = 'block';
            }
            
            // 应用筛选并显示结果
            applyFilter();
        }
        
        // 显示详细信息
        function showDetails(index) {
            const result = allResults[index];
            const detailContent = document.getElementById('detailContent');
            
            // 移除所有行的选中状态
            document.querySelectorAll('#resultsBody tr').forEach(row => row.classList.remove('selected'));
            // 添加当前行的选中状态
            document.querySelector('#resultsBody tr[data-index="' + index + '"]').classList.add('selected');
            
            let html = '';
            
            // 基本信息
            html += '<div class="detail-item"><div class="detail-label">类型</div><div class="detail-value">' + result.keyInfo.type + '</div></div>';
            
            html += '<div class="detail-item"><div class="detail-label">描述</div><div class="detail-value">' + result.keyInfo.description + '</div></div>';
            
            html += '<div class="detail-item"><div class="detail-label">子类型</div><div class="detail-value">' + (result.keyInfo.subType || '-') + '</div></div>';
            
            html += '<div class="detail-item"><div class="detail-label">UUID</div><div class="detail-value">' + (result.keyInfo.uuid || '-') + '</div></div>';
            
            html += '<div class="detail-item"><div class="detail-label">Key</div><div class="detail-value">' + escapeHtml(result.key) + '</div></div>';
            
            html += '<div class="detail-item"><div class="detail-label">Value</div><div class="detail-value">' + escapeHtml(formatValue(result.valueInfo.value)) + '</div></div>';
            
            // Key 详情
            if (Object.keys(result.keyInfo.details).length > 0) {
                const keyDetailsJson = JSON.stringify(result.keyInfo.details, null, 2);
                html += '<div class="detail-item"><div class="detail-label">Key 详情</div><div class="detail-value">' + syntaxHighlight(keyDetailsJson) + '</div></div>';
            }
            
            // Value 详情
            if (Object.keys(result.valueInfo.details).length > 0) {
                const valueDetailsJson = JSON.stringify(result.valueInfo.details, null, 2);
                html += '<div class="detail-item"><div class="detail-label">Value 详情</div><div class="detail-value">' + syntaxHighlight(valueDetailsJson) + '</div></div>';
            }
            
            detailContent.innerHTML = html;
        }
        
        // 格式化值
        function formatValue(value) {
            if (typeof value === 'string' && value.length > 100) {
                return value.substring(0, 100) + '...';
            }
            return value;
        }
        
        // 截断文本
        function truncateText(text, maxLength) {
            if (typeof text === 'string' && text.length > maxLength) {
                return text.substring(0, maxLength) + '...';
            }
            return text;
        }
        
        // 转义 HTML 特殊字符
        function escapeHtml(text) {
            return text
                .replace(/&/g, '&amp;')
                .replace(/</g, '&lt;')
                .replace(/>/g, '&gt;')
                .replace(/"/g, '&quot;')
                .replace(/'/g, '&#039;');
        }
        
        // 格式化详情内容
        function formatDetails(details) {
            if (!details || Object.keys(details).length === 0) {
                return '-';
            }
            
            // 提取第一个键值对作为显示内容
            const keys = Object.keys(details);
            if (keys.length > 0) {
                const firstKey = keys[0];
                const firstValue = details[firstKey];
                let displayValue = firstValue;
                
                // 如果值是对象或数组，转换为字符串
                if (typeof firstValue === 'object' && firstValue !== null) {
                    displayValue = JSON.stringify(firstValue);
                }
                
                // 截断过长的内容
                return truncateText(firstKey + ': ' + displayValue, 20);
            }
            
            return '有';
        }
        
        // JSON 语法高亮
        function syntaxHighlight(json) {
            json = json.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
            return json.replace(/"([^"]+)"\s*:\s*(("[^"]*")|(\d+)|(true|false)|(null))/g, function(match, key, value) {
                let keyClass = 'json-key';
                let valueClass = 'json-string';
                
                if (value === 'true' || value === 'false') {
                    valueClass = 'json-boolean';
                } else if (!isNaN(value) && value !== null) {
                    valueClass = 'json-number';
                } else if (value === 'null') {
                    valueClass = 'json-null';
                }
                
                return '<span class="' + keyClass + '">"' + key + '"</span>: <span class="' + valueClass + '">' + value + '</span>';
            });
        }
        
        // 初始化
        document.addEventListener('DOMContentLoaded', function() {
            // 初始显示 inode tab
            switchTab('inode');
        });
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
