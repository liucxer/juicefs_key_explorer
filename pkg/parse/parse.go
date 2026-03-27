package parse

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"juicefs_key_explorer/pkg/model"
)

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

// GetTypeName 根据类型值获取类型名称
func GetTypeName(t uint8) string {
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

// ParseKey 解析 JuiceFS TiKV key
func ParseKey(key []byte) *model.KeyInfo {
	info := &model.KeyInfo{
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
func parseTypeAndContent(info *model.KeyInfo, content []byte) {
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
func parseAType(info *model.KeyInfo, content []byte) {
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
func parseCType(info *model.KeyInfo, content []byte) {
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
func parseSType(info *model.KeyInfo, content []byte) {
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
func parseDirectString(info *model.KeyInfo, content []byte) {
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
func ParseValue(keyInfo *model.KeyInfo, value []byte) *model.ValueInfo {
	info := &model.ValueInfo{
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
func parseSettingValue(info *model.ValueInfo, value []byte) {
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
func parseCounterValue(info *model.ValueInfo, value []byte) {
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
func parseSessionValue(info *model.ValueInfo, value []byte) {
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
func parseSessionInfoValue(info *model.ValueInfo, value []byte) {
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

// parseNodeValue 解析 Node 类型 value（二进制 Attr）
func parseNodeValue(info *model.ValueInfo, value []byte) {
	info.Type = "Node"
	info.Description = "Binary encoded file attributes"
	info.Value = fmt.Sprintf("0x%x", value)

	// 尝试解析 Attr 结构
	rb := model.NewBufferReader(value)

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
	info.Details["TypeName"] = GetTypeName(typ)
}

// parseEdgeValue 解析 Edge 类型 value（二进制 {type, inode}）
func parseEdgeValue(info *model.ValueInfo, value []byte) {
	info.Type = "Edge"
	info.Description = "Binary encoded {type, inode}"
	info.Value = fmt.Sprintf("0x%x", value)

	if len(value) >= 9 {
		typ := value[0]
		info.Details["Type"] = GetTypeName(typ)
		info.Details["Inode"] = binary.BigEndian.Uint64(value[1:9])
	} else {
		info.Details["RawValue"] = fmt.Sprintf("0x%x", value)
	}
}

// parseChunkValue 解析 Chunk 类型 value（二进制 Slices）
func parseChunkValue(info *model.ValueInfo, value []byte) {
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
func parseSymlinkValue(info *model.ValueInfo, value []byte) {
	info.Type = "Symlink"
	info.Description = "Symlink target path"
	info.Value = string(value)
	info.Details["Target"] = string(value)
}

// parseXattrValue 解析 Xattr 类型 value（xattr value）
func parseXattrValue(info *model.ValueInfo, value []byte) {
	info.Type = "Xattr"
	info.Description = "Extended attribute value"
	info.Value = fmt.Sprintf("0x%x", value)
	info.Details["RawValue"] = string(value)
}

// FormatKey 生成智能显示的 key 字符串
func FormatKey(key []byte) string {
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
