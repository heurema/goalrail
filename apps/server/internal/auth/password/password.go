package password

import (
	"errors"
	"fmt"

	"github.com/alexedwards/argon2id"
)

const (
	memoryKiB   = 19 * 1024
	iterations  = 2
	parallelism = 1
	saltLength  = 16
	keyLength   = 32
)

var (
	ErrEmptyPassword     = errors.New("password is empty")
	ErrMalformedHash     = errors.New("password hash is malformed")
	ErrUnsupportedHash   = errors.New("password hash is unsupported")
	ErrRandomSaltFailure = errors.New("generate password salt")
)

var currentParams = &argon2id.Params{
	Memory:      memoryKiB,
	Iterations:  iterations,
	Parallelism: parallelism,
	SaltLength:  saltLength,
	KeyLength:   keyLength,
}

// HashPassword returns a PHC-style Argon2id password hash.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}
	hash, err := argon2id.CreateHash(password, currentParams)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrRandomSaltFailure, err)
	}
	return hash, nil
}

// VerifyPassword checks password against a PHC-style Argon2id password hash.
func VerifyPassword(password string, encodedHash string) (bool, error) {
	if password == "" {
		return false, ErrEmptyPassword
	}
	match, err := argon2id.ComparePasswordAndHash(password, encodedHash)
	if err != nil {
		return false, mapHashError(err)
	}
	return match, nil
}

// NeedsRehash reports whether encodedHash uses different Argon2id parameters
// than Goalrail's current password-hashing settings.
func NeedsRehash(encodedHash string) (bool, error) {
	params, _, _, err := argon2id.DecodeHash(encodedHash)
	if err != nil {
		return false, mapHashError(err)
	}
	return !sameParams(params, currentParams), nil
}

func sameParams(a *argon2id.Params, b *argon2id.Params) bool {
	return a != nil &&
		b != nil &&
		a.Memory == b.Memory &&
		a.Iterations == b.Iterations &&
		a.Parallelism == b.Parallelism &&
		a.SaltLength == b.SaltLength &&
		a.KeyLength == b.KeyLength
}

func mapHashError(err error) error {
	switch {
	case errors.Is(err, argon2id.ErrInvalidHash):
		return ErrMalformedHash
	case errors.Is(err, argon2id.ErrIncompatibleVariant):
		return ErrUnsupportedHash
	case errors.Is(err, argon2id.ErrIncompatibleVersion):
		return ErrUnsupportedHash
	default:
		return ErrMalformedHash
	}
}
