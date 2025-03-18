package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/cxio/archives/_v0/base"
	"github.com/cxio/archives/_v0/logs"
	"github.com/cxio/archives/_v0/storage"
	"github.com/cxio/archives/_v0/utils"
)

var _T = base.GetText

// Config 配置信息
type Config struct {
	MaxFileSize int64  // 最大上传文件大小
	MaxMetaSize int64  // 最大元信息文件大小
	UILanguage  string // 用户界面语言
}

// DocumentHandler 处理文档的添加和检索
type DocumentHandler struct {
	config  *Config
	store   storage.Documenter
	fmetaer storage.MetaCreator
}

// NewDocumentHandler 创建一个新的文档处理程序
func NewDocumentHandler(config *Config, store storage.Documenter, metaer storage.MetaCreator) *DocumentHandler {
	return &DocumentHandler{
		config:  config,
		store:   store,
		fmetaer: metaer,
	}
}

// HealthHandler 健康检查处理程序
func (h *DocumentHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK!"))
}

// UploadHandler 处理文档上传
func (h *DocumentHandler) UploadHandler(w http.ResponseWriter, r *http.Request) {
	// 只接受 POST 请求
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 检查文件大小
	if r.ContentLength > h.config.MaxFileSize {
		http.Error(w, "File size exceeds limit", http.StatusRequestEntityTooLarge)
		return
	}

	// 获取ContentType以确定文档类型
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 读取请求主体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logs.App.Error("Reading request body:", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 生成文档ID (SHA3:Sum256)
	docID := utils.HashSHA3(body)

	// 设置基础元信息
	// 后续可以通过上传元信息文件进行更新。
	meta := h.fmetaer(&storage.BaseMeta{})
	meta.Type(contentType)
	meta.Title(r.Header.Get("X-Document-Title"))
	meta.Summary(r.Header.Get("X-Document-Summary"))
	meta.Uploader(r.Header.Get("X-Document-Uploader"))
	meta.UploadTime(time.Now().Unix())
	meta.Size(len(body))

	// 保存文档
	year, err := h.store.Store("", docID, body, meta)
	if err != nil {
		logs.App.Error("Store document:", err)
		http.Error(w, "Failed to store document", http.StatusInternalServerError)
		return
	}
	logs.App.Info("Document stored successfully with ID:", docID)

	// 当前页面用语言
	lang := parseAcceptLanguage(r.Header.Get("Accept-Language"), h.config.UILanguage)

	// 返回文档关联信息
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"document_id": docID,
		"message":     _T(lang, "Document stored successfully"),
		"year":        year,
		"doc_size":    fmt.Sprintf("%d", len(body)),
	})
}

// UploadMetaHandler 处理文档元信息单独上传
// 注意文档ID和语言是从URL路径中传递的。
// 如果语言为空，则对应无语言后缀的元信息文件，这会覆盖上传文档时自动创建的元信息版本。
// 这里的元信息作为一个数据对象（JSON）传递。
func (h *DocumentHandler) UploadMetaHandler(w http.ResponseWriter, r *http.Request) {
	// 只接受 PUT 请求
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 检查文件大小
	if r.ContentLength > h.config.MaxMetaSize {
		http.Error(w, "File size exceeds limit", http.StatusRequestEntityTooLarge)
		return
	}

	// 从URL路径中获取文档ID
	vars := mux.Vars(r)
	docID := vars["id"]
	if docID == "" {
		http.Error(w, "Document ID is required", http.StatusBadRequest)
		return
	}
	// 上传Meta内容语言
	// 可为空，对应无语言后缀的元信息文件。
	lang := vars["lang"]

	// 读取请求主体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logs.App.Printf("Error reading request body: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	meta := h.fmetaer(&storage.BaseMeta{})
	// 解码元信息
	if err = meta.Unmarshal(body); err != nil {
		logs.App.Printf("Error decoding metadata: %v", err)
		http.Error(w, "Error decoding metadata", http.StatusBadRequest)
		return
	}

	// 保存元信息
	year, err := h.store.StoreMeta("", docID, meta, lang)
	if err != nil {
		logs.App.Printf("Failed to store document metadata: %v", err)
		http.Error(w, "Failed to store document metadata", http.StatusInternalServerError)
		return
	}
	logs.App.Info("Document metadata stored successfully for ID:", docID)

	// 请求者语言
	// 回显给用户的语言，与上传的元信息语言无关。
	lang = parseAcceptLanguage(r.Header.Get("Accept-Language"), h.config.UILanguage)

	// 返回成功消息
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"document_id": docID,
		"message":     _T(lang, "Document metadata stored successfully"),
		"year":        year,
	})
}

// FetchHandler 检索目标年度的文档
func (h *DocumentHandler) FetchHandler(w http.ResponseWriter, r *http.Request) {
	// 只接受 GET 请求
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL路径中获取文档ID
	vars := mux.Vars(r)
	docID := vars["id"]
	if docID == "" {
		http.Error(w, "Document ID is required", http.StatusBadRequest)
		return
	}

	// 从URL路径中获取年份
	year := vars["year"]
	if year == "" {
		http.Error(w, "Year is required", http.StatusBadRequest)
		return
	}
	// 语言可为空，对应无语言后缀的元信息文件。
	lang := vars["lang"]

	// 获取文档数据
	data, meta, err := h.store.Get(year, docID, lang)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Document not found", http.StatusNotFound)
		} else {
			logs.App.Printf("Failed to retrieve document: %v", err)
			http.Error(w, "Failed to retrieve document", http.StatusInternalServerError)
		}
		return
	}

	// 设置基础元信息
	w.Header().Set("Content-Type", meta.Type(""))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", meta.Size(-1)))

	w.Header().Set("X-Document-ID", docID)
	w.Header().Set("X-Document-Year", year)
	w.Header().Set("X-Document-Title", meta.Title(""))
	w.Header().Set("X-Document-Uploader", meta.Uploader(""))
	w.Header().Set("X-Document-UploadTime", fmt.Sprintf("%d", meta.UploadTime(-1)))

	// 如果非空，添加到响应头中
	summary := meta.Summary("")
	if summary != "" {
		w.Header().Set("X-Document-Summary", summary)
	}

	// 返回文档数据
	w.Write(data)
}

// FetchYearHandler 处理文档检索
// 会从传递的起始年度开始逆向查找，默认逆向回溯10年。
func (h *DocumentHandler) FetchYearHandler(w http.ResponseWriter, r *http.Request) {
	// 只接受 GET 请求
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL路径中获取文档ID
	vars := mux.Vars(r)
	docID := vars["id"]
	if docID == "" {
		http.Error(w, "Document ID is required", http.StatusBadRequest)
		return
	}

	// 语言可为空，对应无语言后缀的元信息文件。
	lang := vars["lang"]

	// 起始年份，无值表示当前年度。
	// 注意：年份从URL查询参数中获取。
	year := r.URL.Query().Get("year")

	// 获取文档数据
	data, meta, year, err := h.store.GetFromYear(year, docID, lang)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Document not found", http.StatusNotFound)
		} else {
			logs.App.Printf("Failed to retrieve document: %v", err)
			http.Error(w, "Failed to retrieve document", http.StatusInternalServerError)
		}
		return
	}

	// 设置响应头中的元信息
	w.Header().Set("Content-Type", meta.Type(""))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", meta.Size(-1)))

	w.Header().Set("X-Document-ID", docID)
	w.Header().Set("X-Document-Year", year)
	w.Header().Set("X-Document-Title", meta.Title(""))
	w.Header().Set("X-Document-Uploader", meta.Uploader(""))
	w.Header().Set("X-Document-UploadTime", fmt.Sprintf("%d", meta.UploadTime(-1)))

	// 如果有概要，添加到响应头
	summary := meta.Summary("")
	if summary != "" {
		w.Header().Set("X-Document-Summary", summary)
	}

	// 返回文档数据
	w.Write(data)
}

// FetchMetaHandler 检索目标年度的文档元信息
// 元信息是作为数据（JSON）返回的。
func (h *DocumentHandler) FetchMetaHandler(w http.ResponseWriter, r *http.Request) {
	// 只接受 GET 请求
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL路径中获取文档ID
	vars := mux.Vars(r)
	docID := vars["id"]
	if docID == "" {
		http.Error(w, "Document ID is required", http.StatusBadRequest)
		return
	}

	// 从URL路径中获取年份
	year := vars["year"]
	if year == "" {
		http.Error(w, "Year is required", http.StatusBadRequest)
		return
	}
	// 语言可为空，对应无语言后缀的元信息文件。
	lang := vars["lang"]

	// 获取文档元信息
	meta, err := h.store.GetMeta(year, docID, lang)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Document metadata not found", http.StatusNotFound)
		} else {
			logs.App.Printf("Failed to retrieve document metadata: %v", err)
			http.Error(w, "Failed to retrieve document metadata", http.StatusInternalServerError)
		}
		return
	}

	// 返回元信息
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meta)
}

// FetchMetaYearHandler 单独检索文档元信息
// 元信息是作为数据（JSON）返回的。
// 会从起始年度开始逆向查找，默认逆向回溯10年。
func (h *DocumentHandler) FetchMetaYearHandler(w http.ResponseWriter, r *http.Request) {
	// 只接受 GET 请求
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL路径中获取文档ID
	vars := mux.Vars(r)
	docID := vars["id"]
	if docID == "" {
		http.Error(w, "Document ID is required", http.StatusBadRequest)
		return
	}
	// 语言可为空，对应无语言后缀的元信息文件。
	lang := vars["lang"]

	// 起始年份，无值表示当前年度。
	year := r.URL.Query().Get("year")

	// 获取文档元信息
	meta, year, err := h.store.GetMetaFromYear(year, docID, lang)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Document metadata not found", http.StatusNotFound)
		} else {
			logs.App.Printf("Failed to retrieve document metadata: %v", err)
			http.Error(w, "Failed to retrieve document metadata", http.StatusInternalServerError)
		}
		return
	}

	// 返回元信息
	w.Header().Set("Content-Type", "application/json")
	// 当前存档年份
	w.Header().Set("X-Document-Year", year)

	json.NewEncoder(w).Encode(meta)
}

// ExistHandler 通过HEAD请求检查目标年度的文档是否存在
// 如果文档存在，返回状态码200，否则404。
func (h *DocumentHandler) ExistHandler(w http.ResponseWriter, r *http.Request) {
	// 只接受 HEAD 请求
	if r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL路径中获取文档ID
	vars := mux.Vars(r)
	docID := vars["id"]
	if docID == "" {
		http.Error(w, "Document ID is required", http.StatusBadRequest)
		return
	}

	// 从URL路径中获取年份
	year := vars["year"]
	if year == "" {
		http.Error(w, "Year is required", http.StatusBadRequest)
		return
	}

	// 检查文档是否存在
	if h.store.Exists(year, docID) {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "Document not found", http.StatusNotFound)
	}
}

// ExistYearHandler 通过HEAD请求检查文档的存在性
// 会从起始年度开始逆向查找。
// 如果文档存在，返回状态码200，同时会在响应头中返回文档存储年份。
func (h *DocumentHandler) ExistYearHandler(w http.ResponseWriter, r *http.Request) {
	// 只接受 HEAD 请求
	if r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL路径中获取文档ID
	vars := mux.Vars(r)
	docID := vars["id"]
	if docID == "" {
		http.Error(w, "Document ID is required", http.StatusBadRequest)
		return
	}

	// 起始年份，无值表示当前年度。
	year := r.URL.Query().Get("year")

	// 检查文档是否存在
	year, existed := h.store.ExistsFromYear(year, docID)
	if !existed {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	// 设置文档存储年份
	w.Header().Set("X-Document-Year", year)

	w.WriteHeader(http.StatusOK)
}

// detectLanguage 检测用户首选语言
func parseAcceptLanguage(acceptLang string, defaultLang string) string {
	if acceptLang == "" {
		return defaultLang
	}

	// 分割多语言声明
	langs := strings.Split(acceptLang, ",")
	if len(langs) == 0 {
		return defaultLang
	}

	// 获取首选语言并去除权重部分
	primaryLang := langs[0]
	if idx := strings.Index(primaryLang, ";"); idx > 0 {
		primaryLang = primaryLang[:idx]
	}

	return strings.TrimSpace(primaryLang)
}
