package fss

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/cxio/archives/logs"
	"github.com/cxio/archives/storage"
)

// DocumentMeta 文档元信息
// 用于实现 DocumentMetaer 接口。
// 注意在调用各个获取方法前，应当先解码导入数据。
type DocumentMeta struct {
	storage.BaseMeta        // 基础元信息
	Language         string `json:"language,omitempty"`    // 主语言
	Author           string `json:"author,omitempty"`      // 源作者
	Copyright        string `json:"copyright,omitempty"`   // 版权信息
	CreateTime       string `json:"create_time,omitempty"` // 原始创建时间
}

// Marshal 编码元信息数据
// 因为元信息主要用于阅读，因此强制使用缩进格式（友好）。
func (dm *DocumentMeta) Marshal() ([]byte, error) {
	return json.MarshalIndent(dm, "", "\t")
}

// Unmarshal 解码元信息数据
func (dm *DocumentMeta) Unmarshal(data []byte) error {
	return json.Unmarshal(data, dm)
}

// NewDocumentMeta 创建新的文档元信息
func NewDocumentMeta(base *storage.BaseMeta) storage.DocumentMetaer {
	return &DocumentMeta{
		BaseMeta: *base,
	}
}

func init() {
	// 注册文档元信息创建器
	storage.RegisterMetaCreator(storage.FileSystem, NewDocumentMeta)
}

//
// 存档实现
//////////////////////////////////////////////////////////////////////////////

// DocumentStore 管理文档的存储和检索
// 回溯年数包含起始年份，因此1年的回溯年数表示只检查当前年份。
type DocumentStore struct {
	lookbackYears  int         // 查找回溯年数
	existbackYears int         // 存在性探查回溯年数
	fs             *FileSystem // 文件系统存档
	cache          *sync.Map   // 缓存文档ID和存档年份
}

// NewDocumentStore 创建新的存档管理器
func NewDocumentStore(rootPath string) (*DocumentStore, error) {
	fs, err := NewFileSystem(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize file system: %w", err)
	}
	return &DocumentStore{
		fs:    fs,
		cache: &sync.Map{},
	}, nil
}

// SetLookbackYears 设置回溯年数
// 应用程序应当在初始化时调用此函数，默认为10年。
// @years 回溯年数
func (ds *DocumentStore) SetLookbackYears(years int) {
	if years <= 0 || years > 1000 {
		logs.Dev.Warn("Invalid lookback years, using default value")
		return

	}
	ds.lookbackYears = years
}

// SetExistbackYears 设置存在性检索的回溯年数
// 应用程序应当在初始化时调用此函数，默认为100年。
// @years 回溯年数
func (ds *DocumentStore) SetExistbackYears(years int) {
	if years <= 0 || years > 1000 {
		logs.Dev.Warn("Invalid existback years, using default value")
		return
	}
	ds.existbackYears = years
}

// Store 存储文档及其元数据
// 自动存储的文档元信息没有语言区分，用户可用StoreMeta存储特定语言的元信息。
// @year 指定年份，空串表示当前年份
// @docID 文档ID，应为文档数据的哈希值
// @data 文档数据
// @meta 文档元信息（未指定语言版本）
// @return1 存档所在年份
// @return2 错误
func (ds *DocumentStore) Store(year, docID string, data []byte, meta storage.DocumentMetaer) (string, error) {
	// 是否已缓存
	if _, ok := ds.cache.Load(docID); ok {
		return "", storage.NewDocError(storage.OPStore, "document already exists")
	}

	// 计算存档路径
	if year == "" {
		year = time.Now().Format("2006")
	}
	paths := ds.calculatePaths(year, docID)

	// 不必重复上传（哈希唯一）
	if ds.fs.Exists(paths.DataPath) {
		return "", storage.NewDocError(storage.OPStore, "document already exists")
	}
	// 存储文档数据
	err := ds.fs.Write(paths.DataPath, data)
	if err != nil {
		return "", fmt.Errorf("failed to write document data: %w", err)
	}

	// 序列化并存档元信息
	metaBytes, err := meta.Marshal()
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// 存储默认语言的元信息
	err = ds.fs.Write(paths.MetaPath, metaBytes)
	if err != nil {
		return "", fmt.Errorf("failed to write metadata: %w", err)
	}

	logs.Data.WithFields(log.Fields{
		"did":  docID,
		"year": year,
	}).Info("Document stored")

	// 缓存文档ID&年份
	ds.cache.Store(docID, year)

	return year, nil
}

// StoreMeta 单独存储特定语言的元信息
// 元信息可以被覆盖。请注意与关联文档的存档年份保持一致。
// @year 指定年份，可为空串，表示当前年份
// @docID 文档ID
// @meta 文档元信息
// @lang 文档语言
// @return1 存档所在年份
// @return2 错误
func (ds *DocumentStore) StoreMeta(year, docID string, meta storage.DocumentMetaer, lang string) (string, error) {
	// 计算存储路径
	if year == "" {
		year = time.Now().Format("2006")
	}
	paths := ds.calculatePaths(year, docID)

	// 先检查文档是否存在，保证一致性。
	if !ds.fs.Exists(paths.DataPath) {
		return "", storage.NewDocError(storage.OPStore, "document not found")
	}

	// 日志记录覆写情况。
	if ds.fs.Exists(paths.MetaPath) {
		logs.Data.WithFields(log.Fields{
			"did":  docID,
			"year": year,
		}).Warn("Document meta file overwritten")
	}

	// 序列化并存储元信息
	metaBytes, err := meta.Marshal()
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}
	langMetaPath := paths.MetaPath

	lang = normalizeLang(lang)
	if lang != "" {
		langMetaPath += "." + lang
	}

	err = ds.fs.Write(langMetaPath, metaBytes)
	if err != nil {
		return "", fmt.Errorf("failed to write metadata: %w", err)
	}

	logs.Data.WithFields(log.Fields{
		"did":  docID,
		"year": year,
		"lang": lang,
	}).Info("Written localized metadata")

	return year, nil
}

// Exists 检查目标年度的文档存在性
// 如果文档不存在，不会回溯。用于快速检索。
// @year 目标年份
// @docID 文档ID
// @return1 文档元信息
// @return2 错误
func (ds *DocumentStore) Exists(year, docID string) bool {
	docID = strings.ToLower(docID)

	if _, ok := ds.cache.Load(docID); ok {
		return true
	}
	return ds.fs.Exists(ds.calculatePaths(year, docID).DataPath)
}

// ExistsFromYear 检查文档的存在性
// @year 回溯起始年份，可为空串，表示当前年份
// @docID 文档ID
// @return1 文档元信息
// @return2 错误
func (ds *DocumentStore) ExistsFromYear(year, docID string) (string, bool) {
	docID = strings.ToLower(docID)

	if y, ok := ds.cache.Load(docID); ok {
		return y.(string), true
	}
	year, err := ds.findDocumentYear(year, docID, ds.existbackYears)

	if err != nil {
		logs.Dev.Error("Find document year failed:", err)
		return "", false
	}
	return year, true
}

// Get 获取目标年度的文档数据和元信息
// 如果文档不存在，直接返回错误（不回溯）。用于快速检索。
// @year 目标年份
// @docID 文档ID
// @lang 文档语言，可选（空串），已标准化
func (ds *DocumentStore) Get(year, docID, lang string) ([]byte, storage.DocumentMetaer, error) {
	docID = strings.ToLower(docID)

	data, err := ds.getDocumentData(year, docID)
	if err != nil {
		logs.Dev.WithFields(log.Fields{
			"did":  docID,
			"year": year,
			"lang": lang,
		}).Warn("Failed to get document data")

		return nil, nil, fmt.Errorf("failed to get document data: %w", err)
	}

	meta, err := ds.getMetaWithYear(year, docID, lang)
	if err != nil {
		logs.Dev.WithFields(log.Fields{
			"did":  docID,
			"year": year,
		}).Warn("Failed to get metadata")

		return data, nil, fmt.Errorf("failed to get document metadata: %w", err)
	}
	return data, meta, nil
}

// GetFromYear 检索文档数据和元信息
// @year 回溯起始年份，可为空串，表示当前年份
// @docID 文档ID
// @lang 文档语言，可为空串（默认语言元信息）
// @return1 文档数据
// @return2 文档元信息
// @return3 实际存档所在年份
// @return4 错误
func (ds *DocumentStore) GetFromYear(year, docID, lang string) ([]byte, storage.DocumentMetaer, string, error) {
	docID = strings.ToLower(docID)
	var err error

	// 优先缓存查找年度
	if y, ok := ds.cache.Load(docID); ok {
		year = y.(string)
	} else {
		year, err = ds.findDocumentYear(year, docID, ds.lookbackYears)
	}
	if err != nil {
		return nil, nil, "", fmt.Errorf("document not found: %w", err)
	}

	data, err := ds.getDocumentData(year, docID)
	if err != nil {
		return data, nil, year, fmt.Errorf("failed to get document data: %w", err)
	}

	// 获取元信息
	meta, err := ds.getMetaWithYear(year, docID, lang)
	if err != nil {
		return data, nil, year, fmt.Errorf("failed to get metadata: %w", err)
	}

	return data, meta, year, nil
}

// GetMeta 获取目标年度的文档元信息
// 如果文档不存在，直接返回错误（不回溯）。用于快速检索。
// 如果特定语言的元信息不存在，返回默认元信息。
// @year 目标年份
// @docID 文档ID
// @lang 文档语言，可选（空串），已标准化
func (ds *DocumentStore) GetMeta(year, docID, lang string) (storage.DocumentMetaer, error) {
	docID = strings.ToLower(docID)

	meta, err := ds.getMetaWithYear(year, docID, lang)
	if err != nil {
		logs.Dev.WithFields(log.Fields{
			"did":  docID,
			"year": year,
		}).Warn("Failed to get metadata")

		return nil, fmt.Errorf("failed to get document metadata: %w", err)
	}
	return meta, nil
}

// GetMetaFromYear 检索文档的元信息
// 如果特定语言的元信息不存在，返回默认元信息。
// @year 回溯起始年份，可为空串，表示当前年份
// @docID 文档ID
// @lang 文档语言，可为空串
// @return1 文档元信息
// @return2 实际存档所在年份
// @return3 错误
func (ds *DocumentStore) GetMetaFromYear(year, docID, lang string) (storage.DocumentMetaer, string, error) {
	docID = strings.ToLower(docID)

	var err error
	if y, ok := ds.cache.Load(docID); ok {
		year = y.(string)
	} else {
		year, err = ds.findDocumentYear(year, docID, ds.lookbackYears)
	}
	if err != nil {
		return nil, "", fmt.Errorf("document not found: %w", err)
	}

	meta, err := ds.getMetaWithYear(year, docID, normalizeLang(lang))
	if err != nil {
		logs.Dev.WithFields(log.Fields{
			"did":  docID,
			"year": year,
		}).Warn("Failed to get metadata")

		return meta, year, fmt.Errorf("failed to get metadata: %w", err)
	}

	return meta, year, nil
}

// Delete 删除文档
// 包含删除文档数据和所有的元信息。
// 注意需要提供准确的存档年份。
// @year 目标年份
// @docID 文档ID
func (ds *DocumentStore) Delete(year, docID string) error {
	paths := ds.calculatePaths(year, docID)

	// 删除文档数据
	err := ds.fs.Remove(paths.DataPath)
	if err != nil {
		return fmt.Errorf("failed to remove document data: %w", err)
	}

	logs.Data.WithFields(log.Fields{
		"did":  docID,
		"year": year,
	}).Info("Document deleted")

	// 删除全部语言的元信息
	ds.deleteMetaAll(paths.DirPath)

	logs.Data.WithFields(log.Fields{
		"did":  docID,
		"year": year,
	}).Info("Document all metadata deleted")

	// 删除目录本身
	err = ds.fs.Remove(paths.DirPath)
	if err != nil {
		return fmt.Errorf("failed to remove document directory: %w", err)
	}

	return nil
}

// DeleteMeta 删除文档元信息
// 主要用于单独删除特定语言的元信息。如果未指定语言，删除默认元信息。
// 注意：需要指定准确的存档年份。
// @year 目标年份
// @docID 文档ID
// @lang 文档语言
func (ds *DocumentStore) DeleteMeta(year, docID, lang string) error {
	paths := ds.calculatePaths(year, docID)

	// 删除元信息
	lang = normalizeLang(lang)
	if lang != "" {
		langMetaPath := fmt.Sprintf("%s.%s", paths.MetaPath, lang)
		err := ds.fs.Remove(langMetaPath)
		if err != nil {
			return fmt.Errorf("failed to remove metadata: %w", err)
		}
	} else {
		err := ds.fs.Remove(paths.MetaPath)
		if err != nil {
			return fmt.Errorf("failed to remove metadata: %w", err)
		}
	}

	logs.Data.WithFields(log.Fields{
		"did":  docID,
		"year": year,
		"lang": lang,
	}).Info("Document metadata deleted")

	return nil
}

// Close 关闭存储器
func (ds *DocumentStore) Close() error {
	return ds.fs.Close()
}

// 查找文档所在的年份
// 回溯查找，直到找到文档或达到最大年数。
// @year 回溯起始年份，空串表示当前年份
// @docID 文档ID
// @maxYears 最大回溯年数（含起始年份）
// @return1 文档所在年份
// @return2 错误
func (ds *DocumentStore) findDocumentYear(year, docID string, maxYears int) (string, error) {
	// 获取当前年份和往前几年，按顺序搜索
	currentYear := time.Now().Format("2006")
	if year != "" {
		currentYear = year
	}

	// 当前年份（起始）
	start, err := strconv.Atoi(currentYear)
	if err != nil {
		return "", fmt.Errorf("invalid year: %w", err)
	}
	for i := 1; i <= maxYears; i++ {
		paths := ds.calculatePaths(currentYear, docID)

		if ds.fs.Exists(paths.DataPath) {
			return currentYear, nil
		}
		currentYear = strconv.Itoa(start - i)
	}

	return "", storage.NewDocError(storage.OPGet, "document not found")
}

// 获取目标年度某文档
// 如果文档不存在，直接返回错误（不回溯）。用于快速检索。
// @year 目标年份
// @docID 文档ID
func (ds *DocumentStore) getDocumentData(year, docID string) ([]byte, error) {
	paths := ds.calculatePaths(year, docID)

	data, err := ds.fs.Read(paths.DataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read document data: %w", err)
	}
	return data, nil
}

// 根据年份和DocID，获取元信息
// @year 目标年份
// @docID 文档ID
// @lang 目标语言，可选（空串），已标准化
func (ds *DocumentStore) getMetaWithYear(year, docID, lang string) (*DocumentMeta, error) {
	paths := ds.calculatePaths(year, docID)

	// 尝试读取特定语言的元信息
	var metaPath string
	if lang != "" {
		langMetaPath := fmt.Sprintf("%s.%s", paths.MetaPath, lang)
		if ds.fs.Exists(langMetaPath) {
			metaPath = langMetaPath
		}
	}

	// 如果没有特定语言版本，使用默认元信息
	if metaPath == "" {
		metaPath = paths.MetaPath
	}

	// 读取元信息文件
	metaBytes, err := ds.fs.Read(metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// 解析元信息
	meta := DocumentMeta{}
	if err = meta.Unmarshal(metaBytes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &meta, nil
}

// deleteMetaAll 删除文档的所有元信息
// @year 目标年份
// @docID 文档ID
func (ds *DocumentStore) deleteMetaAll(dirPath string) error {
	files, err := ds.fs.List(dirPath)
	if err != nil {
		return fmt.Errorf("failed to list metadata: %w", err)
	}
	for _, file := range files {
		if strings.Index(file, ".meta") > 0 {
			err = ds.fs.Remove(filepath.Join(dirPath, file))
			if err != nil {
				return fmt.Errorf("failed to remove metadata: %w", err)
			}
		}
	}
	return nil
}

// DocumentPaths 包含文档的各种路径
type DocumentPaths struct {
	DirPath  string // 文档目录路径
	DataPath string // 文档数据文件路径
	MetaPath string // 文档元信息文件路径
}

// calculatePaths 根据文档ID计算存储路径
// - 从docID提取前三个字节用于分层
// - 末端用文档的全ID构建目录，无重名风险（且便于验证）
// - 从docID提取前16个字节用于文件名
func (ds *DocumentStore) calculatePaths(year string, docID string) DocumentPaths {
	// 错误ID时的默认值
	byte1 := "0xff__"
	byte2 := "FF__"
	byte3 := "255__"

	if len(docID) >= 6 {
		byte1 = "0x" + docID[0:2]
		byte2 = strings.ToUpper(docID[2:4])
		byte3Value, _ := strconv.ParseInt(docID[4:6], 16, 0)
		byte3 = fmt.Sprintf("%03d", byte3Value)
	}

	// 构建存档目录路径（末端全ID）
	dirPath := filepath.Join(year, byte1, byte2, byte3, docID)
	fname := docID
	if n := len(docID); n > 16 {
		fname = docID[:16]
	}

	return DocumentPaths{
		DataPath: filepath.Join(dirPath, fname+".data"),
		MetaPath: filepath.Join(dirPath, fname+".meta"),
		DirPath:  dirPath,
	}
}

// 语言字符串标准化
// e.g. "en-US" -> "en_us"
func normalizeLang(lang string) string {
	if lang == "" {
		return ""
	}
	return strings.ToLower(strings.ReplaceAll(lang, "-", "_"))
}
