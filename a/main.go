package main

import (
	"context"
	"log"
	"main/asr"
	"main/link"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 1. 初始化ASR客户端
	asrClient, err := asr.NewASRClient()
	if err != nil {
		log.Fatalf("初始化ASR客户端失败: %v", err)
	}

	// 2. 设置路由
	http.HandleFunc("/asr-stream", link.HandleWebSocket(asrClient))
	http.Handle("/", http.FileServer(http.Dir("../../static"))) // 前端静态文件
	// 检测是否有效的生成wav文件
	//http.HandleFunc("/play", asr.ServeWAVFile)

	// 3. 配置优雅关闭
	server := &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("服务器启动，监听 %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器错误: %v", err)
		}
	}()

	<-done
	log.Println("服务器正在关闭...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("服务器关闭失败: %v", err)
	}
	log.Println("服务器已关闭")
}
