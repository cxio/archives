package base

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// 本地化文件目录
const localDir = "locale"

// 消息结构体，匹配JSON文件中的结构
type Message struct {
	Text  string `json:"text"`  // 原始消息
	Local string `json:"local"` // 翻译消息
}

// 存储已加载的消息翻译
var translations = make(map[string]map[string]string)
var translationsLock sync.RWMutex
var loadedLanguages = make(map[string]bool)

// GetText 获取指定语言的翻译文本
func GetText(lang, msg string) string {
	// 标准化语言代码
	lang = normalizeLang(lang)

	// 尝试加载翻译文件
	if !loadedLanguages[lang] {
		loadLanguage(lang)
	}

	// 查找翻译
	translationsLock.RLock()
	defer translationsLock.RUnlock()

	if translations[lang] != nil {
		if translated, ok := translations[lang][msg]; ok && translated != "" {
			return translated
		}
	}

	// 如果找不到翻译，则返回原始消息
	return msg
}

// loadLanguage 加载指定语言的翻译文件
func loadLanguage(lang string) {
	translationsLock.Lock()
	defer translationsLock.Unlock()

	// 标记为已尝试加载
	loadedLanguages[lang] = true

	// 确定语言和地区
	langParts := strings.Split(lang, "-")
	langOnly := langParts[0]

	// 构建可能的文件路径
	possiblePaths := []string{
		filepath.Join(localDir, lang+".json"),
	}

	// 如果语言有地区，也尝试加载通用语言文件
	if len(langParts) > 1 {
		possiblePaths = append(possiblePaths,
			filepath.Join(localDir, langOnly+".json"))
	}

	// 尝试每个可能的路径
	for _, path := range possiblePaths {
		loadTranslationFile(lang, path)
	}
}

// loadTranslationFile 从文件加载翻译
func loadTranslationFile(lang, path string) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var messages []Message
	err = json.Unmarshal(file, &messages)
	if err != nil {
		return err
	}

	// 初始化语言的翻译映射
	if translations[lang] == nil {
		translations[lang] = make(map[string]string)
	}

	// 添加每条消息的翻译
	for _, msg := range messages {
		if msg.Local != "" {
			translations[lang][msg.Text] = msg.Local
		}
	}

	return nil
}

// 标准化语言代码
// 输入源已保证符合浏览器语言标准，如 "en-US"、"zh-CN"（非 zh_CN）
func normalizeLang(lang string) string {
	// 文件名约束
	// e.g. "zh-CN" -> "zh-cn"
	lang = strings.ToLower(lang)

	parts := strings.Split(lang, "-")
	if len(parts) == 1 {
		// 只有语言部分，如 "en"
		return parts[0]
	} else {
		// 有语言和地区部分，如 "en-us"
		return parts[0] + "-" + parts[1]
	}
}
