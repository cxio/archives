# 程序构建

首先分析实现程序需要的逻辑结构，如果复杂度较高，可以考虑分模块（目录）规划和创建。

程序的基本要求如下。

## 可配置性

包含一个 config.hjson 文件，可由用户配置一些需要的外部信息。比如服务端口（`serve_port`）。


## 使用帮助

包含一组单独的使用说明文档（usage.md、usage.zh_cn.md, ...），用户在命令行使用 -h 或 --help 时，程序根据用户的本地语言环境，读取相应语言的说明文档显示在标准输出。

说明文档的不同语言版本与操作系统语言的对应关系，通过说明文档（如 usage）的后缀名体现出来，如 usage.zh_cn.md 表示简体中文语言版本的使用说明。

> **注：**
> 文档的语言后缀使用操作系统语言名称的小写形式，语言和地区的分隔符使用下划线（如 zh_CN => zh_cn），忽略编码部分。

命令行使用时还包括用 -v 或 --version 显示版本说明。版本说明被定义在一个单独的文件 version（无扩展名） 中。


## 包含日志

在基础的 logs.go 文件中创建三个全局的日子记录器（log.Loger）：

1. App: 记录程序普通的日志，包括错误、警告和普通信息。
2. Dev: 用于记录开发阶段需要的详细信息（Debug）。
3. Data: 记录数据存储&请求相关的信息。这是日常运行中需要记录的日志，需要按天数切分存储。


## 本地化

创建一个单独的 local 包，里面有一个 i18n.go 源码文件，文件中包含一个 `func GetText(lang, msg string) string` 函数。
函数的意思是：当传入目标语言和消息字符串时，获取对应语言的翻译文本。

该包内包含不同语言的子目录，比如：en、zh 等，子目录内包含该语言不同地区的消息文件，比如 `en-us.json`、`zh-cn.json` 等。
消息文件为JSON格式，结构如下：

```json
[
    {
        "text": "原文消息。作为键，不应有重复",
        "local": "原文的本地化翻译。若与原文语言相同则无需翻译，置为空串或可选。"
    },
    {
        "text": "...",
        "local": "（译文）"
    }
]
```

> **注意：**
> 消息文件名为语言和地区的名称通过短横线连接而成（如 `zh-cn`），扩展名为 `.json`（最终如 `zh-cn.json`）。
