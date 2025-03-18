package storage

import (
	"fmt"
)

const (
	FileSystem = "fss" // FileSystem 文件系统存档
)

// DocError 表示文档存储错误
type DocError struct {
	Type    OpType
	Message string
}

// OpType 定义错误类型
type OpType string

const (
	OPStore  OpType = "store"  // OpStore 存档错误
	OPGet    OpType = "get"    // OPGet 获取错误
	OPDelete OpType = "delete" // OPDelete 删除错误
	OPHead   OpType = "head"   // OPHead 检查错误
)

// Error 实现错误接口
func (e *DocError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewDocumentError 创建新的文档错误
// @name 错误类型
// @msg 错误消息
// @err 上级错误延续
func NewDocError(name OpType, msg string) *DocError {
	return &DocError{
		Type:    name,
		Message: msg,
	}
}

// Documenter 文档存储器接口
// 定义通用文档的存储和检索接口。
//   - 首个参数为存档年份，可为空串，表示当前年份。
//     年份是有意设计的，用于表达存档文件的历史性，同时也方便管理。
//   - 文档ID是文档数据的哈希值，以唯一标识文档。
type Documenter interface {
	// 存储文档及其元数据
	// 自动存储的文档元信息没有语言区分，用户可用StoreMeta存储特定语言的元信息。
	// @year 指定年份，空串表示当前年份
	// @docID 文档ID，应为文档数据的哈希值
	// @data 文档数据
	// @meta 文档元信息（未指定语言版本）
	// @return1 存档所在年份
	// @return2 错误
	Store(year, docID string, data []byte, meta DocumentMetaer) (string, error)

	// StoreMeta 单独存储特定语言的元信息
	// 元信息可以被覆盖。这通常在需要多语言支持时使用。
	// @year 指定年份，可为空串，表示当前年份
	// @docID 文档ID
	// @meta 文档元信息
	// @lang 文档语言
	// @return1 存档所在年份
	// @return2 错误
	StoreMeta(year, docID string, meta DocumentMetaer, lang string) (string, error)

	// SetExistbackYears 设置探查回溯年数
	// 仅用于 ExistsFromYear 的简单探查，通常比较长（比如100年）。
	// @years 查询回溯年数
	SetExistbackYears(years int)

	// Exists 检查目标年度的文档存在性
	// 如果文档不存在，返回错误。用于快速检索。
	// @year 目标年份
	// @docID 文档ID
	// @return 文档是否存在
	Exists(year, docID string) bool

	// ExistsFromYear 检查文档的存在性
	// 按默认配置，回溯的年数为100年。与GetFromYear的10年不同，这个更久远。
	// @year 回溯起始年份，可为空串，表示当前年份
	// @docID 文档ID
	// @return 文档是否存在
	ExistsFromYear(year, docID string) (string, bool)

	// SetLookbackYears 设置查找回溯年数
	// 用于下面带 ...FromYear 的检索方法，通常比较短（比如10年）。
	// @years 查询回溯年数
	SetLookbackYears(years int)

	// Get 获取目标年度的文档数据和元信息
	// 如果文档不存在，直接返回错误（不回溯）。用于快速检索。
	// @year 目标年份
	// @docID 文档ID
	// @lang 文档语言，可选，需已标准化
	// @return1 文档数据
	// @return2 文档元信息
	// @return3 错误
	Get(year, docID, lang string) ([]byte, DocumentMetaer, error)

	// GetFromYear 检索文档数据和元信息
	// @year 回溯起始年份，可为空串，表示当前年份
	// @docID 文档ID
	// @lang 文档语言，可为空串
	// @return1 文档数据
	// @return2 文档元信息
	// @return3 文档实际存档所在年份
	// @return4 错误
	GetFromYear(year, docID, lang string) ([]byte, DocumentMetaer, string, error)

	// GetMeta 获取目标年度的文档元信息
	// 如果文档不存在，直接返回错误（不回溯）。用于快速检索。
	// 如果特定语言的元信息不存在，返回默认元信息。
	// @year 目标年份
	// @docID 文档ID
	// @lang 文档语言，可选，需已标准化
	// @return 文档元信息
	// @return 错误
	GetMeta(year, docID, lang string) (DocumentMetaer, error)

	// GetMetaFromYear 检索文档的元信息
	// 如果特定语言的元信息不存在，返回默认元信息。会回溯查找。
	// @year 回溯起始年份，可为空串，表示当前年份
	// @docID 文档ID
	// @lang 文档语言，可为空串
	// @return1 文档元信息
	// @return2 存档所在年份
	// @return3 错误
	GetMetaFromYear(year, docID, lang string) (DocumentMetaer, string, error)

	// Delete 删除文档
	// 包含删除文档数据和所有的元信息。
	// 年份需要准确指定，不会回溯查找。
	// @year 目标年份
	// @docID 文档ID
	Delete(year, docID string) error

	// DeleteMeta 删除文档元信息
	// 单独删除目标语言的元信息。年份需要准确指定。
	// @year 目标年份
	// @docID 文档ID
	// @lang 文档语言
	DeleteMeta(year, docID, lang string) error

	// Close 关闭存储器
	// 完成可能需要的清理工作。
	Close() error
}

// DocumentMetaer 定义文档元信息接口
type DocumentMetaer interface {
	// Type 文档类型
	// 如果实参非空，则为设置文档类型。返回文档类型。
	Type(string) string

	// Title 文档标题
	// 如果实参非空，则为设置文档标题。返回文档标题。
	Title(string) string

	// Summary 文档摘要
	// 如果实参非空，则为设置文档摘要。返回文档摘要。
	Summary(string) string

	// Uploader 文档上传者
	// 如果实参非空，则为设置文档上传者。返回文档上传者。
	Uploader(string) string

	// UploadTime 文档上传时间
	// 如果实参大于零，则为设置文档上传时间。返回文档上传的时间。
	UploadTime(int64) int64

	// Size 文档大小
	// 如果实参大于零，则为设置文档大小。返回文档大小。
	Size(int) int

	// Marshal 编码元信息数据
	Marshal() ([]byte, error)

	// Unmarshal 解码元信息数据
	Unmarshal(data []byte) error
}

//
// 基础元信息
//////////////////////////////////////////////////////////////////////////////

// BaseMeta 基础元信息
// 具体的文档类型元信息应当嵌入该类型，
// 注：未实现 Marshal/Unmarshal 方法，不满足 DocumentMetaer 接口。
type BaseMeta struct {
	DocType       string `json:"type"`
	DocTitle      string `json:"title"`
	DocSummary    string `json:"summary"`
	DocUploader   string `json:"uploader"`
	DocUploadTime int64  `json:"upload_time"`
	DocSize       int    `json:"size"`
}

// Type 返回文档类型
func (m *BaseMeta) Type(t string) string {
	if t != "" {
		m.DocType = t
	}
	return m.DocType
}

// Title 返回文档标题
func (m *BaseMeta) Title(t string) string {
	if t != "" {
		m.DocTitle = t
	}
	return m.DocTitle
}

// Summary 返回文档摘要
func (m *BaseMeta) Summary(s string) string {
	if s != "" {
		m.DocSummary = s
	}
	return m.DocSummary
}

// Uploader 返回文档上传者
func (m *BaseMeta) Uploader(u string) string {
	if u != "" {
		m.DocUploader = u
	}
	return m.DocUploader
}

// UploadTime 返回文档上传时间
func (m *BaseMeta) UploadTime(t int64) int64 {
	if t > 0 {
		m.DocUploadTime = t
	}
	return m.DocUploadTime
}

// Size 返回文档大小
func (m *BaseMeta) Size(sz int) int {
	if sz > 0 {
		m.DocSize = sz
	}
	return m.DocSize
}

//
// 具体实现注册
//////////////////////////////////////////////////////////////////////////////

// MetaCreator 元信息创建器
// 用于特定存档方式的元信息实例创建。
type MetaCreator func(*BaseMeta) DocumentMetaer

// 文档元信息工厂
// 用于特定存档方式的元信息实例创建。
var metaFactories = make(map[string]MetaCreator)

// GetMetaCreator 获取目标类型的元信息创建器
// @name 文档类型
// @return 元信息创建器
func GetMetaCreator(name string) MetaCreator {
	if creator, ok := metaFactories[name]; ok {
		return creator
	}
	return nil
}

// RegisterMetaCreator 注册元信息创建器
// @name 文档类型
// @creator 元信息创建器
func RegisterMetaCreator(name string, creator MetaCreator) {
	metaFactories[name] = creator
}
