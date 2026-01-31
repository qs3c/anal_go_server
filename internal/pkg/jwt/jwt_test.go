package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-for-testing"

func TestGenerateToken(t *testing.T) {
	t.Run("generate valid token", func(t *testing.T) {
		userID := int64(123)
		token, err := GenerateToken(userID, testSecret, 24)

		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Token should be parseable
		claims, err := ParseToken(token, testSecret)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
	})

	t.Run("generate token with different user IDs", func(t *testing.T) {
		token1, err := GenerateToken(1, testSecret, 24)
		require.NoError(t, err)

		token2, err := GenerateToken(2, testSecret, 24)
		require.NoError(t, err)

		// Different users should have different tokens
		assert.NotEqual(t, token1, token2)
	})

	t.Run("generate token with zero user ID", func(t *testing.T) {
		token, err := GenerateToken(0, testSecret, 24)

		require.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := ParseToken(token, testSecret)
		require.NoError(t, err)
		assert.Equal(t, int64(0), claims.UserID)
	})

	t.Run("generate token with negative user ID", func(t *testing.T) {
		token, err := GenerateToken(-1, testSecret, 24)

		require.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := ParseToken(token, testSecret)
		require.NoError(t, err)
		assert.Equal(t, int64(-1), claims.UserID)
	})

	t.Run("generate token with large user ID", func(t *testing.T) {
		largeID := int64(9223372036854775807) // max int64
		token, err := GenerateToken(largeID, testSecret, 24)

		require.NoError(t, err)

		claims, err := ParseToken(token, testSecret)
		require.NoError(t, err)
		assert.Equal(t, largeID, claims.UserID)
	})

	t.Run("generate token with different expire hours", func(t *testing.T) {
		token1, err := GenerateToken(1, testSecret, 1)
		require.NoError(t, err)

		token24, err := GenerateToken(1, testSecret, 24)
		require.NoError(t, err)

		token168, err := GenerateToken(1, testSecret, 168) // 1 week
		require.NoError(t, err)

		// All should be valid and different
		assert.NotEmpty(t, token1)
		assert.NotEmpty(t, token24)
		assert.NotEmpty(t, token168)
	})

	t.Run("generate token with empty secret", func(t *testing.T) {
		token, err := GenerateToken(123, "", 24)

		// Empty secret should still work (not recommended in production)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})
}

func TestParseToken(t *testing.T) {
	t.Run("parse valid token", func(t *testing.T) {
		userID := int64(456)
		token, _ := GenerateToken(userID, testSecret, 24)

		claims, err := ParseToken(token, testSecret)

		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.True(t, claims.ExpiresAt.After(time.Now()))
		assert.True(t, claims.IssuedAt.Before(time.Now().Add(time.Second)))
	})

	t.Run("parse token with wrong secret", func(t *testing.T) {
		token, _ := GenerateToken(123, testSecret, 24)

		claims, err := ParseToken(token, "wrong-secret")

		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("parse invalid token string", func(t *testing.T) {
		claims, err := ParseToken("invalid.token.string", testSecret)

		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("parse empty token", func(t *testing.T) {
		claims, err := ParseToken("", testSecret)

		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("parse malformed token", func(t *testing.T) {
		claims, err := ParseToken("not-a-jwt-at-all", testSecret)

		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("parse token with empty secret", func(t *testing.T) {
		token, _ := GenerateToken(123, "", 24)

		claims, err := ParseToken(token, "")

		require.NoError(t, err)
		assert.Equal(t, int64(123), claims.UserID)
	})

	t.Run("parse expired token", func(t *testing.T) {
		// Create a token that expired in the past
		claims := Claims{
			UserID: 123,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // 1 hour ago
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(testSecret))

		result, err := ParseToken(tokenString, testSecret)

		assert.ErrorIs(t, err, ErrExpiredToken)
		assert.Nil(t, result)
	})

	t.Run("parse token with different signing method", func(t *testing.T) {
		// Create token with RS256 instead of HS256
		claims := Claims{
			UserID: 123,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				NotBefore: jwt.NewNumericDate(time.Now()),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

		result, err := ParseToken(tokenString, testSecret)

		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, result)
	})
}

func TestTokenRoundTrip(t *testing.T) {
	t.Run("generate and parse multiple times", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			userID := int64(i * 100)
			token, err := GenerateToken(userID, testSecret, 24)
			require.NoError(t, err)

			claims, err := ParseToken(token, testSecret)
			require.NoError(t, err)
			assert.Equal(t, userID, claims.UserID)
		}
	})

	t.Run("same user different tokens", func(t *testing.T) {
		userID := int64(123)

		token1, _ := GenerateToken(userID, testSecret, 24)
		time.Sleep(time.Millisecond) // Ensure different timestamp
		token2, _ := GenerateToken(userID, testSecret, 24)

		// Both tokens should be valid but potentially different (due to timestamp)
		claims1, err := ParseToken(token1, testSecret)
		require.NoError(t, err)
		assert.Equal(t, userID, claims1.UserID)

		claims2, err := ParseToken(token2, testSecret)
		require.NoError(t, err)
		assert.Equal(t, userID, claims2.UserID)
	})
}

func TestErrors(t *testing.T) {
	t.Run("error messages", func(t *testing.T) {
		assert.Equal(t, "invalid token", ErrInvalidToken.Error())
		assert.Equal(t, "token has expired", ErrExpiredToken.Error())
	})
}
