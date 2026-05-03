package password

import (
	"errors"
	"strings"
	"testing"

	"github.com/alexedwards/argon2id"
)

func TestHashPasswordUsesPHCStyleArgon2idFormat(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("hash = %q, want PHC-style argon2id prefix with MVP params", hash)
	}
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Fatalf("hash parts = %d, want 6", len(parts))
	}
	if parts[4] == "" || parts[5] == "" {
		t.Fatalf("hash = %q, want salt and derived key parts", hash)
	}
}

func TestVerifyPasswordAcceptsValidPassword(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	ok, err := VerifyPassword("correct-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !ok {
		t.Fatal("VerifyPassword() = false, want true")
	}
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if ok {
		t.Fatal("VerifyPassword() = true, want false")
	}
}

func TestVerifyPasswordMalformedHashReturnsError(t *testing.T) {
	ok, err := VerifyPassword("password", "not-a-phc-hash")
	if !errors.Is(err, ErrMalformedHash) {
		t.Fatalf("VerifyPassword() error = %v, want ErrMalformedHash", err)
	}
	if ok {
		t.Fatal("VerifyPassword() = true, want false")
	}
}

func TestVerifyPasswordUnsupportedAlgorithmReturnsError(t *testing.T) {
	hash, err := HashPassword("password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	unsupported := strings.Replace(hash, "$argon2id$", "$argon2i$", 1)

	ok, err := VerifyPassword("password", unsupported)
	if !errors.Is(err, ErrUnsupportedHash) {
		t.Fatalf("VerifyPassword() error = %v, want ErrUnsupportedHash", err)
	}
	if ok {
		t.Fatal("VerifyPassword() = true, want false")
	}
}

func TestVerifyPasswordUnsupportedVersionReturnsError(t *testing.T) {
	hash, err := HashPassword("password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	unsupported := strings.Replace(hash, "$v=19$", "$v=16$", 1)

	ok, err := VerifyPassword("password", unsupported)
	if !errors.Is(err, ErrUnsupportedHash) {
		t.Fatalf("VerifyPassword() error = %v, want ErrUnsupportedHash", err)
	}
	if ok {
		t.Fatal("VerifyPassword() = true, want false")
	}
}

func TestNeedsRehashRejectsCurrentParams(t *testing.T) {
	hash, err := HashPassword("password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	needsRehash, err := NeedsRehash(hash)
	if err != nil {
		t.Fatalf("NeedsRehash() error = %v", err)
	}
	if needsRehash {
		t.Fatal("NeedsRehash() = true, want false")
	}
}

func TestVerifyPasswordAcceptsDifferentValidParamsAndNeedsRehash(t *testing.T) {
	hash, err := argon2id.CreateHash("password", &argon2id.Params{
		Memory:      8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})
	if err != nil {
		t.Fatalf("argon2id.CreateHash() error = %v", err)
	}

	ok, err := VerifyPassword("password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !ok {
		t.Fatal("VerifyPassword() = false, want true")
	}
	needsRehash, err := NeedsRehash(hash)
	if err != nil {
		t.Fatalf("NeedsRehash() error = %v", err)
	}
	if !needsRehash {
		t.Fatal("NeedsRehash() = false, want true")
	}
}

func TestHashPasswordUsesDifferentSalts(t *testing.T) {
	first, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword() first error = %v", err)
	}
	second, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword() second error = %v", err)
	}
	if first == second {
		t.Fatal("HashPassword() returned identical hashes for the same password")
	}
}

func TestHashPasswordRejectsEmptyPassword(t *testing.T) {
	_, err := HashPassword("")
	if !errors.Is(err, ErrEmptyPassword) {
		t.Fatalf("HashPassword() error = %v, want ErrEmptyPassword", err)
	}
}
