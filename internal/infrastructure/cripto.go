package infrastructure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

type HMACSigner struct {
	secret []byte
}

func NewHMACSigner(secret string) *HMACSigner {
	return &HMACSigner{secret: []byte(secret)}
}

func (s *HMACSigner) Sign(payload []byte) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
