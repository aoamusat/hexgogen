package token

import (
	"fmt"
	"time"

	"github.com/o1egl/paseto"
	"golang.org/x/crypto/chacha20poly1305"

	"github.com/aoamusat/hexgogen/domain/dto"
)

type PasetoTokenGenerator struct {
	paseto       *paseto.V2
	symmetricKey []byte
	privateKey   string
	publicKey    string
}

func NewPasetoToken(key string) (*PasetoTokenGenerator, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("kindly specify a key of length: %d", chacha20poly1305.KeySize)
	}

	return &PasetoTokenGenerator{
		paseto:       paseto.NewV2(),
		symmetricKey: []byte(key),
	}, nil
}

func (p *PasetoTokenGenerator) GenerateToken(claim dto.TokenClaim, duration time.Duration) (string, error) {
	claim.ExpiresAt = time.Now().Add(duration)
	claim.IssuedAt = time.Now()

	return p.paseto.Encrypt(p.symmetricKey, claim, nil)
}

func (p *PasetoTokenGenerator) DecodeToken(token string) (*dto.TokenClaim, error) {
	var claim dto.TokenClaim

	err := p.paseto.Decrypt(token, p.symmetricKey, &claim, nil)
	if err != nil {
		return nil, err
	}

	if !claim.Valid() {
		return nil, fmt.Errorf("expired token")
	}

	return &claim, nil
}
