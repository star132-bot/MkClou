package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-at-least-32-characters-long"

func TestTokenManager_IssueAndParse(t *testing.T) {
	m := NewTokenManager(testSecret, 15*time.Minute)

	token, exp, err := m.Issue(42, "sess-1")
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), exp, time.Second)

	claims, err := m.Parse(token)
	require.NoError(t, err)
	id, err := claims.MerchantID()
	require.NoError(t, err)
	assert.Equal(t, uint64(42), id)
	assert.Equal(t, "sess-1", claims.SessionID)
}

func TestTokenManager_RejectsExpired(t *testing.T) {
	m := NewTokenManager(testSecret, time.Minute)
	m.now = func() time.Time { return time.Now().Add(-2 * time.Minute) }
	token, _, err := m.Issue(1, "s")
	require.NoError(t, err)

	m.now = time.Now
	_, err = m.Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenManager_RejectsWrongSecret(t *testing.T) {
	token, _, err := NewTokenManager(testSecret, time.Minute).Issue(1, "s")
	require.NoError(t, err)

	_, err = NewTokenManager("another-secret-at-least-32-characters", time.Minute).Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenManager_RejectsAlgNone(t *testing.T) {
	claims := Claims{SessionID: "s", RegisteredClaims: jwt.RegisteredClaims{
		Issuer: issuer, Subject: "1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = NewTokenManager(testSecret, time.Minute).Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenManager_RejectsMissingSessionID(t *testing.T) {
	m := NewTokenManager(testSecret, time.Minute)
	token, _, err := m.Issue(1, "")
	require.NoError(t, err)
	_, err = m.Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestOpaqueTokenAndHash(t *testing.T) {
	a, err := NewOpaqueToken()
	require.NoError(t, err)
	b, err := NewOpaqueToken()
	require.NoError(t, err)

	assert.Len(t, a, 43) // 32 字节 Base64 无填充
	assert.NotEqual(t, a, b)
	assert.Len(t, HashToken(a), 64)
	assert.Equal(t, HashToken(a), HashToken(a))
	assert.NotEqual(t, HashToken(a), HashToken(b))
}
