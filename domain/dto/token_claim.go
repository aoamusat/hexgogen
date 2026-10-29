package dto

import (
	"time"

	"github.com/google/uuid"
)

type TokenClaim struct {
	UserID    uuid.UUID `json:"userID"`
	ExpiresAt time.Time `json:"expiresAt"`
	IssuedAt  time.Time `json:"issuedAt"`
	//Data      T         `json:"data"`
}

func NewTokenClaim(userID uuid.UUID, duration time.Duration) TokenClaim {
	return TokenClaim{
		UserID:    userID,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(duration),
	}
}

func (a TokenClaim) Valid() bool {
	return !time.Now().After(a.ExpiresAt)
}
