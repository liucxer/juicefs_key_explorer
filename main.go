package main

import (
	"flag"
	"log"
	"net/http"

	"juicefs_key_explorer/pkg/frontend"
	"juicefs_key_explorer/pkg/server"
)

func main() {
	// 解析命令行参数
	addr := flag.String("addr", ":8080", "Server address")
	flag.Parse()

	// 设置路由
	server.SetupRoutes()
	// 前端页面
	http.HandleFunc("/", frontend.FrontendHandler)

	// 启动服务器
	log.Printf("Server starting on %s", *addr)
	if err := http.ListenAndServe(*addr, server.EnableCORS(http.DefaultServeMux)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
