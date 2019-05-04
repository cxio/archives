内容类型定义：
type: "[type]/[name][,alias][@name]"  // 全小写

type:
- audio     	// 音频类
- block     	// 区块类，通常指区块链区块
- compression	// 压缩类
- html      	// HTML格式文档
- image 	    // 图片类
- resource  	// 资源文档
- simulation   	// 仿真拟像类
- text      	// 纯文本类，子类为各种语言代码
- video 	    // 视频类

示例：
- type: "html/article"          // Html格式的文章。
- type: "text/html"             // HTML源码，注：指代码本身。
- type: "text/javascript,js"    // JavaScript代码，别名JS。
- type: "text/less@css"         // LESS源码，最终需要转换到CSS使用。
- type: "text/markdown@html"    // Markdown源码，最终需要转换到HTML浏览。
- type: "image/png              // 图片资源，png格式。
- type: "image/jpeg,jpg"        // 图片资源，JPEG格式，别名JPG。
