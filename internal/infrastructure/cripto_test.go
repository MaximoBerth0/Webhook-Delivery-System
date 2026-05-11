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
	signer := NewHMACSigner()
	sig := signer.Sign([]byte("hello"), "my-secret")
	if !strings.HasPrefix(sig, "sha256=") {
		t.Errorf("signature missing sha256= prefix: %q", sig)
	}
}

func TestHMACSigner_Sign_CorrectValue(t *testing.T) {
	secret := "my-secret"
	payload := []byte("hello world")
	signer := NewHMACSigner()
	got := signer.Sign(payload, secret)
	want := expectedSignature(secret, "hello world")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHMACSigner_Sign_DifferentSecrets(t *testing.T) {
	signer := NewHMACSigner()
	payload := []byte("same payload")
	sig1 := signer.Sign(payload, "secret-a")
	sig2 := signer.Sign(payload, "secret-b")
	if sig1 == sig2 {
		t.Error("different secrets should produce different signatures")
	}
}

func TestHMACSigner_Sign_DifferentPayloads(t *testing.T) {
	signer := NewHMACSigner()
	sig1 := signer.Sign([]byte("payload-one"), "my-secret")
	sig2 := signer.Sign([]byte("payload-two"), "my-secret")
	if sig1 == sig2 {
		t.Error("different payloads should produce different signatures")
	}
}

func TestHMACSigner_Sign_Deterministic(t *testing.T) {
	signer := NewHMACSigner()
	payload := []byte("deterministic")
	if signer.Sign(payload, "my-secret") != signer.Sign(payload, "my-secret") {
		t.Error("same input should always produce same signature")
	}
}

func TestHMACSigner_Sign_EmptyPayload(t *testing.T) {
	secret := "my-secret"
	signer := NewHMACSigner()
	got := signer.Sign([]byte{}, secret)
	want := expectedSignature(secret, "")
	if got != want {
		t.Errorf("empty payload: got %q, want %q", got, want)
	}
}

func TestHMACSigner_Verify_Valid(t *testing.T) {
	signer := NewHMACSigner()
	payload := []byte("hello world")
	secret := "my-secret"
	sig := signer.Sign(payload, secret)
	if !signer.Verify(payload, secret, sig) {
		t.Error("expected Verify to return true for valid signature")
	}
}

func TestHMACSigner_Verify_WrongSecret(t *testing.T) {
	signer := NewHMACSigner()
	payload := []byte("hello world")
	sig := signer.Sign(payload, "correct-secret")
	if signer.Verify(payload, "wrong-secret", sig) {
		t.Error("expected Verify to return false for wrong secret")
	}
}

func TestHMACSigner_Verify_TamperedPayload(t *testing.T) {
	signer := NewHMACSigner()
	secret := "my-secret"
	sig := signer.Sign([]byte("original"), secret)
	if signer.Verify([]byte("tampered"), secret, sig) {
		t.Error("expected Verify to return false for tampered payload")
	}
}

func TestHMACSigner_Verify_InvalidSignatureFormat(t *testing.T) {
	signer := NewHMACSigner()
	if signer.Verify([]byte("payload"), "secret", "not-hex!!") {
		t.Error("expected Verify to return false for invalid hex")
	}
}
