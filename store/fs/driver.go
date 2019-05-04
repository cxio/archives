//
// Package fs 本地文件系统存取驱动。
// 驱动名称：“fs”或“filesystem”。
//
// 文档和资源分开对待，便于引用处理。
// 如果资源本身就是主体，而非文档内的引用（如纯视频），则该资源由概要索引。
// （注：作为引用出现的资源，一般出现在html类文档中。）
//
package fs

import (
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cxio/archives/store"
)

//
// 默认配置值。
//
const (
	DataDir = "_data"           // 数据存储根目录
	ResDir  = "_data/resources" // 资源存储根目录
	DocPath = "Y/M/D"           // 文档路径分组
	ResPath = "1/2/3"           // 资源路径分组
)

//
// Driver 驱动器。
//
type Driver struct{}

//
// Conn 驱动连接。
// 返回一个当前驱动的仓储对象。
// 配置解析出错时返回错误。
//
func (Driver) Conn(data store.Config) (store.Storer, error) {
	cfg := Config{
		Root:    DataDir,
		Resdir:  ResDir,
		Docpath: DocPath,
		Respath: ResPath,
	}
	err := json.Unmarshal([]byte(data), &cfg)
	if err != nil {
		return nil, err
	}
	ds := dirs{}
	if err = ds.init(cfg); err != nil {
		return nil, err
	}
	return &Store{ds}, nil
}

//
// Config 配置对象。
//
type Config struct {
	Root    string `json"data"`     // 数据存储根目录。默认“_data”
	Resroot string `json"resource"` // 资源存储根目录。默认“_data/resources”

	// 文档路径格式。如："Y/M/D"
	// 仅支持如下7个标识字符：
	// Y - 4位数年份。如 2016
	// M - 2位数月份。01-12
	// D - 当月2位数日期。01-30|31|28|29
	// h - 当日2位数小时（24小时制）。00-23
	// m - 2位数分钟。00-59
	// s - 2位数秒数。00-59
	// d - 当日在一年中的序数（从1开始）
	// 注：各标识符只能单独使用（斜线分隔），但可任意组合。
	DocDirs string `json:"docdir"`
	// 资源路径格式。如："1/2/3"
	// 数值指代Hash序列串字符下标。无需连续分配。
	// 1 - 序列串首个字符
	// 2 - 序列串第二个字符
	// n - 序列串第n个字符（小于序列串长度）
	ResDirs string `json:"resdir"`
}

//
// 子目录获取器。
// 用于获取特定配置的文档/资源子目录路径。
//
type dirs struct {
	doc docSubs
	res resSubs
	// 根目录
	docRoot string
	resRoot string
}

//
// init 初始解析。
// 安全获取文档/资源根目录。
// 文档/资源子目录配置合法性校验。
// 分组获取句柄赋值，子目录名临时存储空间分配。
//
func (d *dirs) init(cfg Config) error {
	dir, err := filepath.Abs(cfg.Root)
	if err != nil {
		return err
	}
	d.docRoot = dir
	dir, err = filepath.Abs(cfg.Resroot)
	if err != nil {
		return err
	}
	d.resRoot = dir
	if err = d.doc.Parse(cfg.DocDirs); err != nil {
		return err
	}
	if err = d.res.Parse(cfg.ResDirs); err != nil {
		return err
	}
	return nil
}

//
// Dir 获取文档存储目录。
// 平台特定，不包含最后的目录文件分隔线。
//
func (d *dirs) Dir(tm time.Time) string {
	return filepath.Join(d.docRoot, d.doc.Dir(tm))
}

//
// 获取资源存储路径。
// 平台特定，不包含最后的目录文件分隔线。
//
func (d *dirs) Resdir(hash string) string {
	return filepath.Join(d.resRoot, d.res.Dir(hash))
}

//
////////////////////////////////////////////////////////////////////////////////
// 仓储接口实现
//

//
// Store 仓储实现。
//
type Store struct {
	ds dirs
}

//
// Get 获取文件读取句柄。
//
func (*Store) Get(id []byte) (io.Reader, error) {
	//
}

//
// Save 保存文件数据。
//
func (*Store) Save(d io.Reader, id []byte) chan error {
	//
}

//
// Remove 移除目标id文件。
//
func (*Store) Remove(id []byte) chan error {
	//
}

//
// Close 关闭仓储连接。
//
func (*Store) Close() error {
	//
}

//
// Hash 获取文件特定位置区数据Hash。
//
func (*Store) Hash(id []byte, beg, size int) []byte {
	//
}

//
////////////////////////////////////////////////////////////////////////////////
// 文档子目录辅助
//

//
// 文档子目录获取句柄。
//
type docDirer func(tm time.Time) string

//
// 文档子目录构造器。
//
type docSubs struct {
	doc []docDirer
	tmp []string
}

//
// 解析配置串。
// cfg 仅接受“YMDhms”六个标志符，如“Y/M/D”。
//
func (ds *docSubs) Parse(cfg string) error {
	tmp := strings.Split("/")
	ds.doc = make([]docDirer, len(tmp))

	for i, v := range tmp {
		if len(v) != 1 || strings.Index("YMDhms", v) < 0 {
			return errors.New("invalid doc-subs defined")
		}
		ds.doc[i] = docDirHandlers[v]
	}
	ds.tmp = tmp
	return nil
}

//
// 获取文档子目录路径。
//
func (ds *docSubs) Dir(tm time.Time) string {
	for i, fn := range ds.doc {
		ds.tmp[i] = fn(tm)
	}
	return filepath.Join(ds.tmp...)
}

//
// 文档子目录获取句柄集。
//   - Y 4位数年
//   - M 2位数月
//   - D 2位数日
//   - h 2位数时（24小时制）
//   - m 2位数分
//   - s 2位数秒
//   - d 年日序数（1-365|366）
//
var docDirHandlers = map[string]docDirer{
	"Y": func(tm time.Time) string { return tm.Format("2006") },
	"M": func(tm time.Time) string { return tm.Format("01") },
	"D": func(tm time.Time) string { return tm.Format("02") },
	"h": func(tm time.Time) string { return tm.Format("15") },
	"m": func(tm time.Time) string { return tm.Format("04") },
	"s": func(tm time.Time) string { return tm.Format("05") },
	"d": func(tm time.Time) string { return strconv.Itoa(tm.YearDay()) },
}

//
////////////////////////////////////////////////////////////////////////////////
// 资源子目录辅助
//

//
// 资源子目录获取句柄。
//
type resDirer func(hash string) string

//
// 绑定位置的资源子目录获取句柄。
// 注意：资源ID前置一个额外的数字0字符。
// Param:
//  - n 配置位置，从1开始
//
func getResDirer(n int) resDirer {
	return func(id string) string {
		if n > len(id) {
			return ""
		}
		// toLower on [0-9a-zA-Z], |00100000
		return string(id[n] | 0x20)
	}
}

//
// 资源子目录构造器。
//
type resSubs struct {
	res []resDirer
	tmp []string
}

//
// 解析配置串。
// cfg 仅接受数字标志符，如“1/2/3”
//
func (rs *resSubs) Parse(cfg string) error {
	tmp := strings.Split(cfg, "/")
	rs.res = make([]resDirer, len(tmp))

	for i, s := range tmp {
		n, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		if n <= 0 {
			return errors.New("invalid res-subs defined")
		}
		rs.res[i] = getResDirer(n)
	}
	rs.tmp = tmp
	return nil
}

//
// 获取资源子目录路径。
//
func (rs *resSubs) Dir(hash string) string {
	for i, fn := range rs.res {
		rs.tmp[i] = fn(hash)
	}
	return filepath.Join(rs.tmp...)
}

//
////////////////////////////////////////////////////////////////////////////////
// 初始注册
//

func init() {
	dr := Driver{}
	store.Register("fs", &dr)
	store.Register("filesystem", &dr)
}
