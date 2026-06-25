package protocols

import (
	"testing"
)

func TestComputeNativePassword(t *testing.T) {
	// mysql_native_password: SHA1(pass) XOR SHA1(salt + SHA1(SHA1(pass)))
	password := "secret"
	salt := []byte("12345678901234567890") // 20 bytes

	result := computeNativePassword(password, salt)

	if len(result) != 20 {
		t.Fatalf("expected 20 bytes, got %d", len(result))
	}

	// Verify consistency
	result2 := computeNativePassword(password, salt)
	for i := range result {
		if result[i] != result2[i] {
			t.Fatalf("inconsistent at byte %d", i)
		}
	}

	// Different password → different result
	result3 := computeNativePassword("other", salt)
	same := true
	for i := range result {
		if result[i] != result3[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("different password should produce different scramble")
	}
}

func TestComputeNativePassword_EmptyPassword(t *testing.T) {
	result := computeNativePassword("", []byte("12345678901234567890"))
	if result != nil {
		t.Fatalf("expected nil for empty password, got %d bytes", len(result))
	}
}

func TestComputeNativePassword_DifferentSalt(t *testing.T) {
	r1 := computeNativePassword("pass", []byte("aaaaaaaaaaaaaaaaaaaa"))
	r2 := computeNativePassword("pass", []byte("bbbbbbbbbbbbbbbbbbbb"))

	same := true
	for i := range r1 {
		if r1[i] != r2[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("different salt should produce different scramble")
	}
}
