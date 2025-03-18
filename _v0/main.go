package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	// 文件系统存档
	fss "github.com/cxio/archives/v0/storage/fss"

	"github.com/cxio/archives/v0/api"
	"github.com/cxio/archives/v0/config"
	"github.com/cxio/archives/v0/logs"
	"github.com/cxio/archives/v0/storage"
)

// 版本信息
const verinfo = "Archives Storage Service v0.1.0"

// 最大上传大小
const (
	minPieceSize = 200 << 20            // 200MB 最小分片大小
	maxFileSize  = 2<<30 + minPieceSize // 2.2GB 文档数据
	maxMetaSize  = 10 << 20             // 10MB 元数据
)

// 文档元信息创建器
// 注：采用文件系统存档方式。
var metaFactory = storage.GetMetaCreator(storage.FileSystem)

var (
	showVersion = flag.Bool("version", false, "Show version information")
	showHelp    = flag.Bool("help", false, "Show help information")
	configFile  = flag.String("config", "./config.hjson", "Path to config file")
	v           = flag.Bool("v", false, "Show version information")
	h           = flag.Bool("h", false, "Show help information")
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 解析命令行参数
	flag.Parse()

	// 别名处理
	if *v {
		*showVersion = true
	}
	if *h {
		*showHelp = true
	}

	// 显示版本
	if *showVersion {
		fmt.Println(verinfo)
		return
	}

	// 配置加载
	cfg, err := config.Init(*configFile)
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// 获取UI语言
	uiLang := detectLanguage(cfg.Language)

	// 显示帮助
	if *showHelp {
		showHelpInfo(uiLang)
		return
	}

	// 升级日志系统
	if err := logs.InitLogs(cfg.LogRootPath, logs.Level(cfg.LogLevel)); err != nil {
		logrus.Fatalf("Failed to initialize logging: %v", err)
	}
	defer logs.CleanupLogs()

	logger := logs.App

	// 启动服务段
	logger.Printf("Starting Archives Storage Service on port %d", cfg.ServePort)

	// 使用文件系统存档
	store, err := fss.NewDocumentStore(cfg.StorageRootPath)
	if err != nil {
		logger.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// 设置2个回溯年数
	store.SetLookbackYears(cfg.FindMaxYears)
	store.SetExistbackYears(cfg.HeadMaxYears)

	// 路由配置
	mux := api.Router(store, metaFactory, &api.Config{
		UILanguage:  uiLang,
		MaxFileSize: maxFileSize,
		MaxMetaSize: maxMetaSize,
	})

	// 启动服务
	errChan := make(chan error, 1)
	go func() {
		errChan <- startAPIServer(ctx, cfg, mux, logger)
	}()

	// 等待退出信号
	select {
	case <-sigChan:
		logger.Info("Shutting down gracefully...")
	case err := <-errChan:
		logger.Errorf("Server error: %v", err)
	}
}

// startAPIServer 启动API服务
func startAPIServer(ctx context.Context, cfg *config.Config, mux *mux.Router, logger *logrus.Logger) error {
	logger.Printf("API server starting on port %d", cfg.ServePort)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.ServePort),
		Handler:           mux,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       60 * time.Second,
	}

	// 优雅关闭服务
	go func() {
		<-ctx.Done()
		logger.Info("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Errorf("Server shutdown error: %v", err)
		}
	}()

	return server.ListenAndServe()
}

// 显示帮助信息
func showHelpInfo(uilang string) {
	// 文件名约束
	// e.g. "zh-CN" -> "zh_cn"
	lang := strings.ToLower(uilang)
	lang = strings.ReplaceAll(lang, "-", "_")

	// 尝试查找对应语言版本的帮助文件
	helpFiles := []string{
		fmt.Sprintf("docs/usage.%s.md", lang),
		"docs/usage.md", // 默认版本
	}

	for _, file := range helpFiles {
		if data, err := os.ReadFile(file); err == nil {
			fmt.Println(string(data))
			return
		}
	}

	// 如果找不到帮助文件，显示简短帮助信息
	fmt.Println("Help file not found. Please check your installation.")
	flag.PrintDefaults()
}

// 检测用户使用的语言
// 如果未指定，则从环境变量中提取，如 "en_US.UTF-8" => "en-US"
// 最终的格式符合如 "en-US"、"zh-CN"，符合标准
func detectLanguage(ulang string) string {
	if ulang == "" {
		lang := os.Getenv("LANG")
		if lang == "" {
			return "en"
		}
		ulang = strings.Split(lang, ".")[0]
	}
	// 下划线替换为连字符
	return strings.ReplaceAll(ulang, "_", "-")
}
