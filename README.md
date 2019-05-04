# 档案存储微服务（archives）

这里的档案存储不是广义上的P2P文档自由存储，而是指区块链上交易里所包含的附件，这些附件并不存储在区块链上（交易中只有一个附件ID），而是存储在archives公共服务网络中，这是区块链世界里的一个基础服务。


## 文档索引与元信息

文档索引用于检索交易附件文档本身，采用附件ID的数据哈希部分（去掉末尾4字节的大小信息），文档类型和一些相关的元信息另外存储，通常为文本描述形式。从P2P离散存储的角度来说，文档数据和它的元信息可以不必在一起，虽然它们通常会在一起。

```go
文件名：[索引ID].*      // 文档数据，索引ID即为数据的哈希摘要
元信息：[索引ID].meta   // 文档元信息，文本格式，可能为多语言
```

这里只实现一个简单的分级存储系统，采用索引ID的逐字节分级（十六进制），前置深度序号（0-V）。初期规模可能为4层分级，如果文件太多分层不够，末端可即时扩展。

```go
_data/                              // 存储根
    000/                            // 一级目录
    ...                             // 首字节16进制值目录名
    0FF/                            // 同级最后一个目录
        100/
        ...
        1FF/                        // 二级目录
            200/
            ...
            2FF/                    // 三级目录
                300/
                ...
                3FF/                // 末端目录
                    ...
                    [索引ID].*      // 文档文件
                    [索引ID].meta   // 文档元信息文件
```


## 固化存储

档案的逻辑是不可修改，只有写入和检索读取，甚至删除都不应该有，通常也是长期保存。因此对于档案存储，我们可以简化设计：

1. 仅包含读取和添加两种逻辑，没有修改和删除的操作。
2. 缩小存储规模仅能通过转储实现：读取 >> 过滤 >> 添加到新仓库。
3. 没有删除操作是一种实用性考虑，以获得一种有意的「不方便」约束。

因为没有修改和删除，这类似于一种固化存储（如并不高效的光盘刻录），它能带来一些难得的优点：

- 便于优化存储，提高数据库效率。
- 没有修改就没有覆写，仅单次写入的优势可能发展出廉价的存储介质，使得大规模存储更易行。
- 安全性更有保障。因为若是单次写入，除非物理上的破坏，不存在覆写丢失的问题。
- 因为简单和安全，在维护成本上也会有更好的表现。
