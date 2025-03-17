package fss

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/cxio/archives/logs"
	"github.com/cxio/archives/storage"
)

// FileSystem 提供文件系统操作
type FileSystem struct {
	rootDir string
}

// NewFileSystem 创建一个新的文件系统管理器
func NewFileSystem(rootDir string) (*FileSystem, error) {
	// 确保根目录存在
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create root directory: %w", err)
	}

	return &FileSystem{
		rootDir: rootDir,
	}, nil
}

// Write 将数据写入指定路径
// @path 文件路径（相对于根目录）
// @data 文件数据
func (fs *FileSystem) Write(path string, data []byte) error {
	// 确保目录存在
	path = fs.filePath(path)
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	logs.Dev.WithFields(log.Fields{
		"path": path,
		"size": len(data),
	}).Debug("File written")

	return nil
}

// Read 从指定路径读取数据
func (fs *FileSystem) Read(path string) ([]byte, error) {
	// 检查文件是否存在
	path = fs.filePath(path)

	if !fs.Exists(path) {
		return nil, storage.NewDocError(storage.OPGet, fmt.Sprintf("file not found: %s", path))
	}

	// 读取文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	logs.Dev.WithFields(log.Fields{
		"path": path,
		"size": len(data),
	}).Debug("File read")

	return data, nil
}

// Exists 检查文件是否存在
func (fs *FileSystem) Exists(path string) bool {
	_, err := os.Stat(
		fs.filePath(path),
	)
	return err == nil || !os.IsNotExist(err)
}

// Remove 删除指定路径的文件
func (fs *FileSystem) Remove(path string) error {
	// 检查文件是否存在
	path = fs.filePath(path)

	if !fs.Exists(path) {
		return storage.NewDocError(storage.OPDelete, fmt.Sprintf("file not found: %s", path))
	}

	// 删除文件
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	logs.Dev.WithField("path", path).Debug("File removed")

	return nil
}

// List 列出目录中的文件
func (fs *FileSystem) List(dirPath string) ([]string, error) {
	// 检查目录是否存在
	dirPath = fs.filePath(dirPath)

	if !fs.Exists(dirPath) {
		return nil, storage.NewDocError(storage.OPHead, fmt.Sprintf("directory not found: %s", dirPath))
	}

	// 读取目录内容
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// 提取文件名
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// 构建文件的完整路径。
// @subs 递进路径子序列
// @return 完整存档路径
func (fs *FileSystem) filePath(path string) string {
	return filepath.Join(fs.rootDir, path)
}

// Close 关闭文件系统
func (fs *FileSystem) Close() error {
	return nil
}
