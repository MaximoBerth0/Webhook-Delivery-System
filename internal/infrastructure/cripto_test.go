package infrastructure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// run the test:
// go test ./internal/infrastructure/ -v -run TestHMACSigner

func expectedSignature(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestHMACSigner_Sign_Prefix(t *testing.T) {
	signer := NewHMACSigner("my-secret")
	sig := signer.Sign([]byte("hello"))
	if !strings.HasPrefix(sig, "sha256=") {
		t.Errorf("signature missing sha256= prefix: %q", sig)
	}
}

func TestHMACSigner_Sign_CorrectValue(t *testing.T) {
	secret := "my-secret"
	payload := []byte("hello world")
	signer := NewHMACSigner(secret)
	got := signer.Sign(payload)
	want := expectedSignature(secret, "hello world")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHMACSigner_Sign_DifferentSecrets(t *testing.T) {
	payload := []byte("same payload")
	sig1 := NewHMACSigner("secret-a").Sign(payload)
	sig2 := NewHMACSigner("secret-b").Sign(payload)
	if sig1 == sig2 {
		t.Error("different secrets should produce different signatures")
	}
}

func TestHMACSigner_Sign_DifferentPayloads(t *testing.T) {
	signer := NewHMACSigner("my-secret")
	sig1 := signer.Sign([]byte("payload-one"))
	sig2 := signer.Sign([]byte("payload-two"))
	if sig1 == sig2 {
		t.Error("different payloads should produce different signatures")
	}
}

func TestHMACSigner_Sign_Deterministic(t *testing.T) {
	signer := NewHMACSigner("my-secret")
	payload := []byte("deterministic")
	if signer.Sign(payload) != signer.Sign(payload) {
		t.Error("same input should always produce same signature")
	}
}

func TestHMACSigner_Sign_EmptyPayload(t *testing.T) {
	secret := "my-secret"
	signer := NewHMACSigner(secret)
	got := signer.Sign([]byte{})
	want := expectedSignature(secret, "")
	if got != want {
		t.Errorf("empty payload: got %q, want %q", got, want)
	}
}
