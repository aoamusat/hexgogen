package token

import (
	"fmt"
	"testing"
	"time"

	"github.com/aoamusat/hexgogen/domain/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewPasetoToken(t *testing.T) {
	testData := []struct {
		name   string
		key    string
		errStr string
	}{
		{
			name:   "can create new Paseto with a Valid symmetric Key",
			key:    "1234567890-qwertyuoip-1234560sds",
			errStr: "<nil>",
		}, {
			name:   "cannot create new Paseto with an Incomplete symmetric Key length",
			key:    "1234567890-qwertyuoip-1",
			errStr: "kindly specify a key of length: 32",
		}, {
			name:   "can create new Paseto when an empty symmetric key is passed",
			key:    "",
			errStr: "kindly specify a key of length: 32",
		},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			pasetoT, err := NewPasetoToken(td.key)
			outputStr := fmt.Sprintf("%v", err)

			assert.Equal(t, td.errStr, outputStr)

			if pasetoT != nil {
				assert.Equal(t, td.key, string(pasetoT.symmetricKey))
			}
		})
	}
}

func TestPasetoTokenGenerator_GenerateAndVerifyToken(t *testing.T) {
	t.Run("can generate valid token and verify", func(t *testing.T) {
		key := "1234567890-qwertyuoip-1234560sds"

		pToken, err := NewPasetoToken(key)
		assert.Nil(t, err)
		claim := dto.TokenClaim{
			UserID: uuid.New(),
		}

		token, err := pToken.GenerateToken(claim, time.Hour)
		assert.Nil(t, err)
		assert.NotEmpty(t, token)

		t.Run("can verify generated token", func(t *testing.T) {
			verifiedClaim, err := pToken.DecodeToken(token)
			assert.Nil(t, err)

			assert.Equal(t, claim.UserID, verifiedClaim.UserID)
		})
	})
}
