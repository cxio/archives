package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/hjson/hjson-go"
)

// 默认配置
const (
	defaultDataDir  = "./data"         // 数据根目录
	defaultLogDir   = "./_logs"        // 日志根目录
	defaultLogLevel = "info"           // 日志级别
	lookbackYears   = 10               // 默认查找年数（回溯）
	existbackYears  = 100              // 默认存在性探查回溯年数
	maxReadTimeout  = 10 * time.Minute // 读取超时，支持大文件配置时间较长
	maxWriteTimeout = 10 * time.Minute // 写入超时，同上
)

// Config 存档应用配置
type Config struct {
	ServePort       int           `json:"serve_port"`        // 服务端口
	StorageRootPath string        `json:"storage_root_path"` // 存档根目录
	Language        string        `json:"language"`          // 用户界面语言
	LogLevel        string        `json:"log_level"`         // 日志级别
	LogRootPath     string        `json:"log_root_path"`     // 日志根目录
	FindMaxYears    int           `json:"find_max_years"`    // 搜索文档时的最大回溯年数
	HeadMaxYears    int           `json:"head_max_years"`    // 检查存在性的最大回溯年数
	ReadTimeout     time.Duration `json:"read_timeout"`      // 读取超时，0值时采用内部默认值
	WriteTimeout    time.Duration `json:"write_timeout"`     // 写入超时，同上
}

// Load 从文件加载配置
// @path 配置文件路径
func Load(path string) (*Config, error) {
	configFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = hjson.Unmarshal(configFile, &config)
	if err != nil {
		return nil, err
	}

	// 设置默认值
	if config.ServePort == 0 {
		config.ServePort = 8080
	}
	if config.StorageRootPath == "" {
		config.StorageRootPath = defaultDataDir
	}
	if config.LogLevel == "" {
		config.LogLevel = defaultLogLevel
	}
	if config.LogRootPath == "" {
		config.LogRootPath = defaultLogDir
	}
	if config.FindMaxYears <= 0 {
		config.FindMaxYears = lookbackYears
	}
	if config.HeadMaxYears <= 0 {
		config.HeadMaxYears = existbackYears
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = maxReadTimeout
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = maxWriteTimeout
	}

	// 扩展可能的用户主目录路径
	config.StorageRootPath = ExpandPath(config.StorageRootPath)
	config.LogRootPath = ExpandPath(config.LogRootPath)

	return &config, nil
}

// Init 初始化配置
// @path 配置文件路径
func Init(path string) (*Config, error) {
	path = ExpandPath(path)

	config, err := Load(path)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// ExpandPath 扩展用户主目录路径
func ExpandPath(path string) string {
	if path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // 获取失败则返回原路径
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
