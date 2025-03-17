package logs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	// App 普通程序级消息
	App *logrus.Logger

	// Dev 开发阶段信息（Debug）
	Dev *logrus.Logger

	// Data 和数据相关的日志，按日期分割
	Data *logrus.Logger

	// 局部使用的变量
	////////////////////////////////////////////////////////////

	// 普通日志文件
	archFile *os.File

	// 调试日志文件
	debugFile *os.File

	// 数据日志记录器
	dataLoger *dataLogWriter
)

// Level 定义日志级别类型
type Level string

const (
	// LevelDebug 调试级别
	LevelDebug Level = "debug"
	// LevelInfo 信息级别
	LevelInfo Level = "info"
	// LevelWarn 警告级别
	LevelWarn Level = "warn"
	// LevelError 错误级别
	LevelError Level = "error"
)

var initialized bool
var initMutex sync.Mutex

// InitLogs 初始化日志系统
// 创建日志文件是时出错，会使用标准输出。以维持系统持续运行。
// @rootDir 日志文件根目录
func InitLogs(rootDir string, level Level) error {
	initMutex.Lock()
	defer initMutex.Unlock()

	if initialized {
		return nil
	}
	// 创建日志目录
	err := os.MkdirAll(rootDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// 创建普通日志文件
	archFile, err = os.OpenFile(filepath.Join(rootDir, "arch.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Error("Failed to open arch log file:", err)
		logrus.Warn("The logging will use standard output")
		archFile = nil
	}

	// 创建调试日志文件
	debugFile, err = os.OpenFile(filepath.Join(rootDir, "debug.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Error("Failed to open debug log file:", err)
		logrus.Warn("The debug logging will use standard output")
		debugFile = nil
	}

	// 按日期创建数据日志文件
	currentDate := time.Now().Format("20060102")
	dataLogFile := openDataLogFile(rootDir, currentDate)

	// 初始化普通日志
	App = logrus.New()
	if archFile == nil {
		App.SetOutput(os.Stdout)
	} else {
		App.SetOutput(io.MultiWriter(os.Stdout, archFile))
	}
	App.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "006-01-02T15:04:05Z07:00",
	})

	// 初始化调试日志
	Dev = logrus.New()
	if debugFile == nil {
		Dev.SetOutput(os.Stdout)
	} else {
		Dev.SetOutput(io.MultiWriter(os.Stdout, debugFile))
	}
	Dev.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "006-01-02T15:04:05Z07:00",
	})

	// 初始化数据日志，使用自定义 Writer 实现日期分割
	Data = logrus.New()
	dataLoger = &dataLogWriter{
		curDate: currentDate,
		logDir:  rootDir,
		logFile: dataLogFile,
		mu:      &sync.Mutex{},
	}
	Data.SetOutput(dataLoger)
	Data.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05", // 数据日志只需要时间
	})

	// 默认设置所有日志为Info级别
	SetLogLevel(level)

	App.Info("日志系统初始化完成")
	initialized = true

	return nil
}

// SetLogLevel 设置所有日志记录器的日志级别
func SetLogLevel(level Level) {
	var logLevel logrus.Level
	switch level {
	case LevelDebug:
		logLevel = logrus.DebugLevel
	case LevelInfo:
		logLevel = logrus.InfoLevel
	case LevelWarn:
		logLevel = logrus.WarnLevel
	case LevelError:
		logLevel = logrus.ErrorLevel
	default:
		logLevel = logrus.InfoLevel
	}

	App.SetLevel(logLevel)
	Dev.SetLevel(logLevel)
	Data.SetLevel(logLevel)
}

// 用于处理数据日志按日期分割的自定义Writer
type dataLogWriter struct {
	curDate string      // 当前日期
	logDir  string      // 日志文件路径
	logFile *os.File    // 当前日志文件
	mu      *sync.Mutex // 用于保护文件操作的互斥锁
}

func (w *dataLogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查日期是否变化，如果变化则创建新的日志文件
	date := time.Now().Format("20060102")
	if date != w.curDate {
		// 关闭旧的日志文件
		if w.logFile != nil {
			if err := w.logFile.Close(); err != nil {
				App.WithError(err).Error("关闭旧数据日志文件失败")
			}
		}

		// 创建新的日志文件
		w.logFile = openDataLogFile(w.logDir, date)
		w.curDate = date
		App.WithField("date", w.curDate).Info("数据日志文件已切换")
	}

	// 写入到控制台
	if _, err := os.Stdout.Write(p); err != nil {
		return 0, err
	}

	// 如果文件打开失败，会仅使用标准输出，维持运行
	if w.logFile == nil {
		return 0, nil
	}
	return w.logFile.Write(p)
}

func (w *dataLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.logFile != nil {
		return w.logFile.Close()
	}
	return nil
}

// 打开指定日期的数据日志文件
func openDataLogFile(rootDir, date string) *os.File {
	// 样式：data_20250302.log
	filename := filepath.Join(rootDir, fmt.Sprintf("data_%s.log", date))

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// 记录到普通日志
		App.Error("Failed to open data log file:", err)
		App.Warn("The data logging will use standard output")
		return nil
	}
	return file
}

// CleanupLogs 关闭日志文件
func CleanupLogs() {
	if archFile != nil {
		App.Info("日志系统关闭")
		if err := archFile.Close(); err != nil {
			fmt.Printf("关闭日志文件失败: %v\n", err)
		}
		archFile = nil
	}
	if debugFile != nil {
		Dev.Info("调试日志系统关闭")
		if err := debugFile.Close(); err != nil {
			fmt.Printf("关闭调试日志文件失败: %v\n", err)
		}
		debugFile = nil
	}
	if dataLoger != nil {
		Data.Info("数据日志系统关闭")
		if err := dataLoger.Close(); err != nil {
			fmt.Printf("关闭数据日志文件失败: %v\n", err)
		}
		dataLoger = nil
	}
}
