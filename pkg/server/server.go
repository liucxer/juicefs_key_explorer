package server

import (
	"encoding/json"
	"net/http"

	"juicefs_key_explorer/pkg/model"
	"juicefs_key_explorer/pkg/parse"

	"github.com/tikv/client-go/v2/config"
	"github.com/tikv/client-go/v2/txnkv"
)

// EnableCORS 处理 CORS
func EnableCORS(next http.Handler) http.Handler {
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

// ScanHandler 处理扫描请求
func ScanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.ScanRequest
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
	var results []model.ScanResult
	count := 0

	for iter.Valid() {
		count++
		key := iter.Key()
		value := iter.Value()

		// 解析 key
		keyInfo := parse.ParseKey(key)

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
		valueInfo := parse.ParseValue(keyInfo, value)

		// 生成智能显示的 key 字符串
		formattedKey := parse.FormatKey(key)

		// 添加到结果列表
		results = append(results, model.ScanResult{
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

// HealthHandler 健康检查
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// SetupRoutes 设置路由
func SetupRoutes() {
	http.HandleFunc("/api/scan", ScanHandler)
	http.HandleFunc("/api/health", HealthHandler)
}
