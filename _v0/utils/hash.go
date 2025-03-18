package utils

import (
	"encoding/hex"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"

	"github.com/cxio/archives/_v0/logs"
)

// HashSHA3 计算数据的SHA3-256哈希值并返回十六进制字符串
func HashSHA3(data []byte) string {
	// 创建哈希对象
	hash := sha3.New256()

	// 写入数据
	hash.Write(data)

	// 计算哈希值
	hashBytes := hash.Sum(nil)

	// 转换为十六进制字符串
	hashStr := hex.EncodeToString(hashBytes)

	logs.Dev.WithFields(log.Fields{
		"hash": hashStr[:10] + "...",
		"size": len(data),
	}).Debug("Hash calculated")

	return hashStr
}

// ValidateHash 检查哈希值是否匹配
func ValidateHash(data []byte, expectedHash string) bool {
	// 计算实际哈希值
	actualHash := HashSHA3(data)

	// 比较哈希值
	if actualHash != expectedHash {
		logs.Dev.WithFields(log.Fields{
			"expected": expectedHash,
			"actual":   actualHash,
		}).Debug("Hash validation failed")

		return false
	}

	logs.Dev.WithField("hash", expectedHash[:10]+"...").Debug("Hash validation successful")
	return true
}
