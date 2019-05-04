//
// Package data 通用数据接口定义。
// 主要考虑交互的数据采用JSON格式。
//
package data

import (
	"crypto/sha1"
	"io"
	"sort"
	"time"
)

//
// Types 返回当前可用的类型名清单。
//
func Types() []string {
	var list []string
	for name := range types {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

//
// Hash 固定算法的Hash摘要。
// 用于计算一项内容的id标识，它们在其概要中需要。
// 这是一个耗时的操作！
//
func Hash(r io.Reader) string {
	//
}

//
// FieldJSON JSON字段对象。
// 用于表达Go结构中字段映射到JSON对象内的条目。
// View名称表达js数据的视觉感受，以便良好展示表单控件。
//
type FieldJSON struct {
	// 键名
	Name string
	// 值的JS类型名。
	// 见“T...”系常量定义。
	Type string
	// 视觉大小。
	// 用于表单text类控件UI表现，见“V...”系常量定义。
	View string
}

//
// FieldJSON 结构字段取值定义。
// 关于date类型：
//  - date类型对应到Go里的time.Time类型。
//  - JSON里被表示为标准的格式字符串，合规的格式才能正确转换。
//  - json包默认的时间序列化格式可以被js正确解析。
//
const (
	// .Type
	Tstring  = "string"  // 字符串，单行文本
	Ttext    = "text"    // 多行文本（字符串）
	Tnumber  = "number"  // 数值，包含整数和浮点数
	Tboolean = "boolean" // 布尔值，可用复选框表达
	Tdate    = "date"    // 日期，规范的格式字符串（参考Date.toJSON）
	Tarray   = "array"   // 数组，对应Go里的 []interface{}
	Tobject  = "object"  // 对象，对应Go里的 map[string]interface{}
	Tnull    = "null"    // 对应Go里的 nil

	// .View
	Vfull   = "full"   // 满尺寸。自适应填充容器元素（～100%）
	Vphrase = "phrase" // 中等尺寸。如一个短语，一般仅用于单行
	Vshort  = "short"  // 较短尺寸。单行如一个单词，多行比默认尺寸稍小
	Vmini   = "mini"   // 小尺寸。单行1-2个字符，多行指最小文本框
	Vomit   = ""       // 不指定。不适用或默认尺寸
)

//
// Schemaor 概要接口。
// 概要基本为只读逻辑，只有极少几个字段可修改，并且不影响概要标识（ID）。
//
type Schemaor interface {
	// 概要id
	// 按固定的规则进行计算。
	ID() string
	// 内容id（base58-hash）
	CID() string

	// 源解析
	Parse(json []byte) (Schemaor, error)
	// 编码为JSON
	JSON() ([]byte, error)
	// 数据字段集
	// 用于外部构建JSON数据录入条目。
	Fields() []FieldJSON

	// 基本信息集
	// 不同的类型可有不同的条目集（key）
	Basic() map[string]string
}

//
// NewSchema 创建一个特定类型的概要对象。
// 类型名应当已被注册（子包内的init...），否则抛出panic。
//
// 新的概要对象是空的，各字段的赋值可以采用直接赋值或解析JSON实现。
// 因为不同的类型会有不同的字段扩展，故一般是由外部预先构造JSON解析导入。
// 注：Fields的返回值详细说明了不同类型的字段情况。
//
// 如果内容被修改，应当对概要内的CHash字段更新（Update）。
//
func NewSchema(typeName string) Schemaor {
	op, ok := types[typeName]
	if !ok {
		panic(typeName + " type is invalid.")
	}
	return op.Schema()
}

//
// SchemaFields 基本概要字段集。
// 与Schema类型里Tag-json的名称相同。
//
var SchemaFields = []FieldJSON{
	{"title", Tstring, Vfull},
	{"stakeid", Tstring, Vomit},
	{"hash", Tstring, Vomit},
	{"createtime", Tdate, Vomit},
	{"digest", Ttext, Vfull},
}

//
// Schema 基本概要。
// 具体的数据类型应当嵌入本定义。
// 本类型并未完全实现Schemaor接口，因此不能单独使用。
//
type Schema struct {
	Title    string    `json:"title"`            // 主标题
	Stake    string    `json:"stakeid"`          // 权益ID，区块链如BTC/PPC地址，利益直达
	Created  time.Time `json:"createtime"`       // 创建时间（即内容登记时间）
	CHash    string    `json:"hash"`             // 内容ID（sha256-base58，前置“0”）
	Encrypts string    `json:"encrypts"`         // 加密标识（空值表示未加密）
	Digest   string    `json:"digest,omitempty"` // 内容简介
	Extra    []string  `json:"extra,omitempty"`  // 附属数据id集
	// 不在概要中设置“评级”的说明：
	// 1. 可信度存疑。若包含评级实体信息则可能导致利益性的竞争，并使问题复杂化。
	// 2. 概要最终存储在区块链上，存储需要支付权益，这已经蕴含了一种价值判断。
	// 3. 这种价值判断是大众性的即时行为，拥有历史性价值，而评级可能造成误导，如果节点作偏向性选择存储，则可能损害系统本身。
	// 4. 评级所蕴含的利益力量，还会损害公平性——主流引导对边缘忽略的伤害——它们并非不重要。
	// Star   int       `json:"star,omitempty"`       // 评级（星等）
}

//
// ID 计算概要id。
// 会强制清理标题文本内的空白字符：合并为一个空格。
// Note:
// - 与初始ID可能不同——如果主标题和创建时间改变的话。
//
func (s *Schema) ID() string {
	ts := util.CleanText(s.Title)
	return schemaID(ts, s.CTime)
}

//
// CID 返回内容id（base58）
//
func (s *Schema) CID() string {
	return s.CHash
}

//
// Contenter 实体内容接口。
// 封装了数据获取和加密转换，数据获取从store包得到。
//
type Contenter interface {
	// 获取数据。
	// - 这是一个新的内容对象首先需要调用的操作。
	// - 该操作从store中获取内容，store封装了本地存储和对外部节点的数据请求。
	// Param:
	// - r 新内容读取器，如果为nil则用id从store检索获取。
	// - k 加密标识串，空串表示未加密（或内部分析文件头）。
	Load(r io.Reader, k string) error
	// 返回数据。
	// 可能设置了加密转换。
	Data() io.Reader
	// 设置加密标识
	// - 指示新的加密方式，空串表示不加密；
	// - 返回接口本身便于链式调用；
	Encrypts(flag string) Contenter
}

//
// NewContent 创建一个新的内容对象。
// 每个内容对象对应一个唯一的ID，该id也是内容数据的Hash摘要。
// 如果传递cid为空串，表示新建一个内容对象（尚未存储），此时Load参数应当非nil。
//
func NewContent(cid string) Contenter {
	return &content{id: cid}
}

//
// 内容对象
// - id即为Data内容的Hash摘要；
// - id = Schema.CHash
//
type content struct {
	id      string
	created time.Time
	kold    string // 原加密标识
	knew    string // 新加密标识
	nocrypt bool   // 不加密
}

//
// 载入内容数据。
// 如果r为nil，则表示用id检索已经存储的数据。
// 传递r有效值，表示新载入内容或替换原始内容。
//
func (c *content) Load(r io.Reader) error {
	//
}

//
// 返回内容数据。
// 根据加密标识，数据已经过加密或不加密处理。
//
func (c *content) Data() io.Reader {
	//
}

//
// 设置加密标识。
// 指示某种加密方式，空串表示不加密。
//
func (c *content) Encrypts(flag string) Contenter {
	if flag == "" {
		c.nocrypt = true
	} else {
		c.knew = flag
	}
	return c
}

//
// Extraser 附属数据接口。
// 用于增强概要表达的一些简短内容。
// 对于文章，可能是精彩段落摘抄或精彩评论。
// 对于视频，可能是局部内容剪辑的海报预览。
//
// 附加数据可以添加、删除，但不能编辑修改。
// 它作为单独的数据与内容分开管理，以便于灵活的操作。如内容保密，但想开放部分信息。
//
type Extraser interface {
	// 添加一个数据。
	// 返回当前缓存附加数据的总项数。
	Add(d Extra) int
	// 删除一个数据。
	// 参数为附加数据自身的id（Extra.Id），返回剩余总项数。
	Del(id string) int
	// 获取特定id的数据。
	Get(id string) Extra
	// 将内容刷新到存储。
	// 参数函数用于成功/失败时回调。传递已经成功刷新的id集和是否失败（error非nil）。
	Flush(cb func([]string, error))
	// 获取附属数据id清单。
	// 清单包含所有附属数据，按其时间戳排序。
	List() []string
}

//
// NewExtras 获取目标内容的附属数据集对象。
//
func NewExtras(cid string) Extraor {
	if cid == "" {
		panic("need content id for a new extra.")
	}
	return &extras{cid: cid}
}

//
// Extra 附属数据。
// 与内容的逻辑类似，数据id也为数据本身的Hash摘要，以拥有对数据的确定性。
//
type Extra struct {
	ID      string
	Created time.Time
	Data    io.Reader
}

//
// 附属数据对象。
//
type extras struct {
	// content.id
	cid string
	// 文章类型时可能为精彩摘录或评价；
	// 视频类型时可能为一组预告/海报视频剪辑；
	list []Extra
}

func (e *extras) Add(d Extra) int {
	//
}

func (e *extras) Del(id string) int {
	//
}

func (e *extras) Get(id string) Extra {

}

func (e *extras) Flush(callback func(ok string, err error)) {
	//
}

func (e *extras) List() []string {
	//
}

//
// 创建概要ID。
// 算法：(ctime + title) => sha1 => base58。
// Note: SHA1 hash algorithm as defined in RFC 3174.
//
// Param:
// 	- title 主标题，最好对多余的空白进行了清理
// 	- ctime 内容的创建时间，固定值（非修改时间）
//
func schemaID(title string, ctime time.Time) string {
	// 设置到0时区。
	tm := ctime.UTC().Format("20060102150405")

	// [时间]-[标题]
	sh := sha1.Sum([]byte(tm + "-" + title))
	return base58.Encode(sh[:])
}

//
////////////////////////////////////////////////////////////////////////////////
// 类型注册&提取
//

//
// Getter 概要对象生成器。
// 由类型子包实现并注册到该类型。
//
type Getter interface {
	// 获取概要对象
	Schema() Schemaor
}

//
// 生成器存储。
//
var types = make(map[string]Getter)

//
// Register 数据类型登记。
// 可以多个名称对应到一种操作，如：image和picture都指图片。
//
func Register(name string, proxy Getter) {
	if proxy == nil {
		panic("data Type getter is nil.")
	}
	if _, dup := types[name]; dup {
		panic(name + " already exists in data.types.")
	}
	types[name] = proxy
}
