//
// Package store 仓库存取操作定义。
// 仓库的实际存储由外部配置的服务提供，以驱动名区分。
//
package store

import (
	"io"
	"sort"
)

//
// Drivers 列出支持的存储驱动清单。
//
func Drivers() []string {
	var list []string
	for name := range drivers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

//
// Config 仓库配置对象。
// 不同的仓库驱动大都有不同的配置，因此配置是一个JSON序列。
//
type Config string

//
// Open 打开仓库。
// 相同的驱动类型打开的是同一个仓库，但创建的是一个新的连接。
// 配置解析出错会返回一个错误。
// 试图使用一个不存在的驱动，会抛出panic。
//
func Open(driverName string, cfg Config) (Storer, error) {
	dr, ok := drivers[driverName]
	if !ok {
		panic(driverName + " driver not exist with store.")
	}
	op, err := dr.Conn(cfg)
	if err != nil {
		return nil, err
	}
	return op, nil
}

//
// Driver 仓储驱动器。
// 提供一个具体配置下的仓库连接。
//
type Driver interface {
	Conn(cfg Config) (Storer, error)
}

//
// Storer 仓储接口。
// 因id为内容的哈希摘要，内容改变就会导致id改变，故不支持内容修改操作。
// 仅支持获取、存储和删除，若有修改则只能作为新内容添加。
// 由具体的驱动实现。
//
type Storer interface {
	// 获取目标id的内容读取器。
	Get(id []byte) (io.Reader, error)

	// 保存内容，索引采用参数传递的id。
	// 返回的管道用于保存成功或失败后进行通知（成功传递nil）。
	Save(d io.Reader, id []byte) chan error

	// 删除目标id的内容。
	Remove(id []byte) chan error

	// 关闭连接
	Close() error

	// 局部数据摘要。
	// 主要用于目标文件的存在证明。
	// - 定位的目标数据片段不应超出整体数据本身，超出返回nil；
	// - 传递beg值0，size值-1可以重新验算文件Hash；
	// - 零长度片段返回nil；
	// Param:
	// - id   目标文件hash值
	// - beg  数据片段起点偏移
	// - size 数据片段长度，-1表示余下全部
	Hash(id []byte, beg, size int) []byte
}

//
////////////////////////////////////////////////////////////////////////////////
// 驱动注册
//

var drivers = make(map[string]Driver)

//
// Register 注册存储驱动
// 名称如：fs, mongodb, redis, sqlite等
//
func Register(name string, driver Driver) {
	if driver == nil {
		panic("store driver is nil.")
	}
	if _, dup := drivers[name]; dup {
		panic(name + " already exists in store.drivers.")
	}
	drivers[name] = driver
}
