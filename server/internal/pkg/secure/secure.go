// Package secure 提供敏感数据的对称加密（AES-256-GCM）与哈希（PRD SEC-06）。
//
// 密文格式：nonce(12 字节) || ciphertext || tag(16 字节)。
// 主密钥来自配置（环境变量），不入库、不入代码仓库；密钥版本号随密文一起存库，以支持密钥轮换。
package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

// ErrDecrypt 表示密文被篡改、密钥错误或格式不正确。
var ErrDecrypt = errors.New("decrypt failed")

// Box 持有当前主密钥。
type Box struct {
	version int16
	aead    cipher.AEAD
	hmacKey []byte
}

// NewBox 根据 Base64 编码的 32 字节主密钥创建加密器。
func NewBox(base64Key string, version int16) (*Box, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("master key must be 32 bytes in base64")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// HMAC 密钥由主密钥派生，与加密密钥用途隔离
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("mkclou-hmac-v1"))
	return &Box{version: version, aead: aead, hmacKey: mac.Sum(nil)}, nil
}

// Version 返回当前主密钥版本，写库时一并保存。
func (b *Box) Version() int16 { return b.version }

// Encrypt 加密明文。aad（附加认证数据）用于把密文绑定到具体记录，例如 "shop:42"，
// 防止把一条记录的密文复制到另一条记录后仍能解密。
func (b *Box) Encrypt(plaintext, aad []byte) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return b.aead.Seal(nonce, nonce, plaintext, aad), nil
}

// Decrypt 解密并校验完整性。
func (b *Box) Decrypt(ciphertext, aad []byte) ([]byte, error) {
	n := b.aead.NonceSize()
	if len(ciphertext) < n+b.aead.Overhead() {
		return nil, ErrDecrypt
	}
	out, err := b.aead.Open(nil, ciphertext[:n], ciphertext[n:], aad)
	if err != nil {
		return nil, ErrDecrypt
	}
	return out, nil
}

// HMAC 返回带密钥的 SHA-256 摘要（十六进制），用于加密数据的去重与查找（如卡密）。
func (b *Box) HMAC(data []byte) string {
	m := hmac.New(sha256.New, b.hmacKey)
	m.Write(data)
	return hex.EncodeToString(m.Sum(nil))
}
