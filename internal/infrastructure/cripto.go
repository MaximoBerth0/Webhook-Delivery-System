package infrastructure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type HMACSigner struct{}

func NewHMACSigner() *HMACSigner {
	return &HMACSigner{}
}

func (s *HMACSigner) Sign(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (s *HMACSigner) Verify(payload []byte, secret, signature string) bool {
	expected := s.Sign(payload, secret)
	got := strings.TrimPrefix(signature, "sha256=")
	exp := strings.TrimPrefix(expected, "sha256=")
	gotBytes, err := hex.DecodeString(got)
	if err != nil {
		return false
	}
	expBytes, err := hex.DecodeString(exp)
	if err != nil {
		return false
	}
	return hmac.Equal(gotBytes, expBytes)
}
