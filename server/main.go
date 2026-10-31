package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jigong/internal/api"
	"jigong/internal/db"
)

// Version 由编译时 -ldflags 注入
var Version = "dev"

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	var (
		port    = flag.String("port", env("JIGONG_PORT", "8910"), "监听端口")
		dataDir = flag.String("data-dir", env("JIGONG_DATA_DIR", "./data"), "数据目录")
		webDir  = flag.String("web-dir", env("JIGONG_WEB_DIR", "../web/dist"), "前端资源目录")
	)
	flag.Parse()

	adminUser := env("JIGONG_ADMIN_USER", "admin")
	adminPass := env("JIGONG_ADMIN_PASSWORD", "admin123")

	database, err := db.Open(*dataDir)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer database.Close()

	srv := api.NewServer(database, *webDir)
	if err := srv.EnsureAdmin(adminUser, adminPass); err != nil {
		log.Fatalf("初始化管理员失败: %v", err)
	}

	httpServer := &http.Server{
		Addr:              ":" + *port,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("记工 v%s 服务已启动: http://0.0.0.0:%s  (数据目录: %s, 前端目录: %s)", Version, *port, *dataDir, *webDir)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("关闭异常: %v", err)
	}
	log.Println("服务已停止")
}
