package postgres

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	plaintext := []byte(`{"api_key": "sk_live_test123"}`)

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext should not equal plaintext")
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptDifferentNonce(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	plaintext := []byte("same input")

	c1, _ := Encrypt(plaintext, key)
	c2, _ := Encrypt(plaintext, key)

	if bytes.Equal(c1, c2) {
		t.Fatal("encrypting same plaintext should produce different ciphertext (random nonce)")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	rand.Read(key1)
	rand.Read(key2)

	ciphertext, _ := Encrypt([]byte("secret"), key1)

	_, err := Decrypt(ciphertext, key2)
	if err == nil {
		t.Fatal("decrypt with wrong key should fail")
	}
}

func TestEncryptInvalidKeyLength(t *testing.T) {
	_, err := Encrypt([]byte("data"), []byte("short"))
	if err == nil {
		t.Fatal("should reject non-32-byte key")
	}
}

func TestDecryptInvalidKeyLength(t *testing.T) {
	_, err := Decrypt([]byte("data"), []byte("short"))
	if err == nil {
		t.Fatal("should reject non-32-byte key")
	}
}

func TestDecryptTooShort(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	_, err := Decrypt([]byte("tiny"), key)
	if err == nil {
		t.Fatal("should reject ciphertext shorter than nonce size")
	}
}

func TestEncryptEmptyInput(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	ciphertext, err := Encrypt([]byte{}, key)
	if err != nil {
		t.Fatalf("encrypt empty: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("decrypt empty: %v", err)
	}

	if len(decrypted) != 0 {
		t.Fatalf("expected empty, got %q", decrypted)
	}
}
