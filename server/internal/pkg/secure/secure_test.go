package secure

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newKey(t *testing.T) string {
	t.Helper()
	k := make([]byte, 32)
	_, err := rand.Read(k)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(k)
}

func TestEncryptDecrypt(t *testing.T) {
	b, err := NewBox(newKey(t), 1)
	require.NoError(t, err)

	c1, err := b.Encrypt([]byte("secret"), []byte("shop:1"))
	require.NoError(t, err)
	c2, err := b.Encrypt([]byte("secret"), []byte("shop:1"))
	require.NoError(t, err)
	assert.NotEqual(t, c1, c2, "相同明文每次加密结果不同（随机 nonce）")

	p, err := b.Decrypt(c1, []byte("shop:1"))
	require.NoError(t, err)
	assert.Equal(t, "secret", string(p))
}

func TestDecrypt_RejectsTamperingWrongAADAndWrongKey(t *testing.T) {
	b, _ := NewBox(newKey(t), 1)
	c, _ := b.Encrypt([]byte("secret"), []byte("shop:1"))

	_, err := b.Decrypt(c, []byte("shop:2"))
	assert.ErrorIs(t, err, ErrDecrypt, "密文不能挪用到其他记录")

	tampered := append([]byte{}, c...)
	tampered[len(tampered)-1] ^= 1
	_, err = b.Decrypt(tampered, []byte("shop:1"))
	assert.ErrorIs(t, err, ErrDecrypt, "密文被篡改")

	other, _ := NewBox(newKey(t), 2)
	_, err = other.Decrypt(c, []byte("shop:1"))
	assert.ErrorIs(t, err, ErrDecrypt, "主密钥错误")

	_, err = b.Decrypt([]byte("short"), nil)
	assert.ErrorIs(t, err, ErrDecrypt)
}

func TestNewBox_RejectsBadKey(t *testing.T) {
	_, err := NewBox("not-base64!", 1)
	assert.Error(t, err)
	_, err = NewBox(base64.StdEncoding.EncodeToString([]byte("too-short")), 1)
	assert.Error(t, err)
}

func TestHMAC(t *testing.T) {
	k := newKey(t)
	a, _ := NewBox(k, 1)
	b, _ := NewBox(newKey(t), 1)
	assert.Equal(t, a.HMAC([]byte("CARD-1")), a.HMAC([]byte("CARD-1")))
	assert.NotEqual(t, a.HMAC([]byte("CARD-1")), a.HMAC([]byte("CARD-2")))
	assert.NotEqual(t, a.HMAC([]byte("CARD-1")), b.HMAC([]byte("CARD-1")), "不同密钥结果不同，无法离线撞库")
}
