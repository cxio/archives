# 档案存储服务

基于哈希索引的文档存储和检索服务。哈希算法使用 `SHA3:Sum256`，检索使用64个十六进制字符串作为其ID（兼容大小写）。


## 使用方法

```
archived [选项]
```


## 选项

-h, --help      显示帮助信息
-v, --version   显示版本信息
--config 文件   指定配置文件路径（默认：./config.hjson）


## API接口

- `POST /api/document`
  上传新文档。会返回存档的文档ID（`SHA3:Sum256`），以及存档的年度（当前年份）。

- `PUT /api/meta/{id}/{lang}`
  上传文档的元信息。无lang指定时，会覆盖上传文档时自动生成的默认元信息版本。

- `GET /document/{year}/{id}`
  通过ID检索文档。如果未找到，返回错误消息。
  这应当是你已确知文档存在于该年度时，否则可以使用下面的URL形式。

- `GET /document/{id}?year=yyyy`
  通过ID检索文档。可指定一个起始搜索年度（可选，默认为当前年度）。默认会向后（以前）*十年*的范围内查找。
  返回数据的同时，会在响应头的 `X-Document-Year` 字段上设置实际存档所在年度。

- `GET /meta/{year}/{id}/{lang}`
  通过文档ID获取文档的元数据。如果未找到，返回错误。
  这应当是你已确知目标路径（年度）正确时使用，否则应当使用下面的URL请求。

- `GET /meta/{id}/{lang}?year=yyyy`
  通过文档ID获取文档元数据，可以指定元信息的语言版本。同上可指定一个起始搜索年度，真实存档年度会通过 `X-Document-Year` 字段标记。

- `HEAD /document/{year}/{id}`
  检查目标年度的目标ID的文档是否存在。如果存在，返回状态码200，否则返回404错误。

- `HEAD /document/{id}?year=yyyy`
  从指定的年度逆向逐年检查文档是否存在，未指定起始年度时，从当前年度开始。
  默认会向后检查**100年**的范围，并通过响应头中的 `X-Document-Year` 字段设置实际存档的年度。如果未找到，返回404错误。


## 配置

可以通过 config.hjson 配置参数，详见配置文件内容。


## 示例

上传文档:
```
curl -X POST -F "file=@document.pdf" http://localhost:8080/api/document
```

检索文档:
```
curl http://localhost:8080/document/[document_id] -o document.pdf
```
