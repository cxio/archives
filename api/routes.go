package api

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/cxio/archives/logs"
	"github.com/cxio/archives/storage"
)

const (
	// 语言：
	// 地区不区分大小写，如 zh-CN、zh-cn、en_US...
	reLang = `{lang:([a-z]{2}([_-][a-zA-Z]{2})?)?}`

	// ID：64位哈希值，不区分大小写
	reID = `{id:^[0-9a-fA-F]{64}$}`

	// 年份：4位数字
	reYear = `{year:^\d{4}$}`
)

// Router 设置API路由
func Router(store storage.Documenter, metaer storage.MetaCreator, config *Config) *mux.Router {
	router := mux.NewRouter()

	// 创建处理程序
	docHandler := NewDocumentHandler(config, store, metaer)

	// 上传文档
	router.HandleFunc(
		// POST /api/document
		"/api/document",
		docHandler.UploadHandler).Methods(http.MethodPost)
	// 上传元数据
	router.HandleFunc(
		// POST /api/meta/{id}/{lang}
		strings.Join([]string{`/api/meta`, reID, reLang}, "/"),
		docHandler.UploadMetaHandler).Methods(http.MethodPost)

	// 下载文档
	router.HandleFunc(
		// GET /document/{year}/{id}
		strings.Join([]string{`/document`, reYear, reID}, "/"),
		docHandler.FetchHandler).Methods(http.MethodGet)
	// ?year=xxxx
	router.HandleFunc(
		// GET /document/{id}?year=yyyy
		strings.Join([]string{`/document`, reID}, "/"),
		docHandler.FetchYearHandler).Methods(http.MethodGet)

	// 下载元数据
	router.HandleFunc(
		// GET /meta/{year}/{id}/{lang}
		strings.Join([]string{`/meta`, reYear, reID, reLang}, "/"),
		docHandler.FetchMetaHandler).Methods(http.MethodGet)
	// ?year=xxxx
	router.HandleFunc(
		// GET /meta/{id}/{lang}?year=yyyy
		strings.Join([]string{`/meta`, reID, reLang}, "/"),
		docHandler.FetchMetaYearHandler).Methods(http.MethodGet)

	router.HandleFunc(
		//HEAD /document/{year}/{id}
		strings.Join([]string{`/document`, reYear, reID}, "/"),
		docHandler.ExistHandler).Methods(http.MethodHead)

	router.HandleFunc(
		// HEAD /document/{id}?year=yyyy
		strings.Join([]string{`/document`, reID}, "/"),
		docHandler.ExistYearHandler).Methods(http.MethodHead)

	// 健康检查端点
	router.HandleFunc(
		// GET /health
		"/health", docHandler.HealthHandler).Methods(http.MethodGet)

	logs.Dev.Info("API routes configured")

	return router
}
