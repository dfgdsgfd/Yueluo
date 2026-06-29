package security

import "testing"

func TestHashPasswordUsesArgon2idAndVerifies(t *testing.T) {
	hash, err := HashPassword("secret-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !IsArgon2idHash(hash) {
		t.Fatalf("hash = %q, want Argon2id PHC hash", hash)
	}
	if !VerifyPassword("secret-password", hash) {
		t.Fatal("VerifyPassword rejected the original password")
	}
	if VerifyPassword("wrong-password", hash) {
		t.Fatal("VerifyPassword accepted the wrong password")
	}
}

func TestRandomPasswordLength(t *testing.T) {
	password, err := RandomPassword(16)
	if err != nil {
		t.Fatalf("RandomPassword() error = %v", err)
	}
	if len(password) != 16 {
		t.Fatalf("RandomPassword length = %d, want 16", len(password))
	}
}
