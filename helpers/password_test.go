package helpers

import "testing"

func TestHashPassword_ProducesVerifiableHash(t *testing.T) {
	password := "s3cr3t-pass"

	hash := HashPassword(password)

	if hash == "" {
		t.Fatal("HashPassword returned an empty string")
	}
	if hash == password {
		t.Fatal("HashPassword returned the plaintext password")
	}
	if !VerifyPassword(hash, password) {
		t.Fatal("VerifyPassword failed for the matching password")
	}
}

func TestHashPassword_ProducesDifferentHashesForSameInput(t *testing.T) {
	password := "same-password"

	h1 := HashPassword(password)
	h2 := HashPassword(password)

	// bcrypt uses a random salt, so hashes for the same input must differ.
	if h1 == h2 {
		t.Fatal("expected different hashes for the same password (salt should differ)")
	}
	// Both hashes must still verify against the original password.
	if !VerifyPassword(h1, password) || !VerifyPassword(h2, password) {
		t.Fatal("both hashes should verify against the original password")
	}
}

func TestVerifyPassword_RejectsWrongPassword(t *testing.T) {
	hash := HashPassword("correct-password")

	if VerifyPassword(hash, "wrong-password") {
		t.Fatal("VerifyPassword accepted an incorrect password")
	}
}

func TestVerifyPassword_RejectsInvalidHash(t *testing.T) {
	if VerifyPassword("not-a-valid-bcrypt-hash", "anything") {
		t.Fatal("VerifyPassword accepted an invalid hash")
	}
}
