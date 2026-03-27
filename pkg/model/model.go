package model

import (
	"encoding/binary"
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
