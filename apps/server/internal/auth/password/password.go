package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	algorithm   = "argon2id"
	memoryKiB   = 19 * 1024
	timeCost    = 2
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

type params struct {
	memory      uint32
	time        uint32
	parallelism uint8
}

var encoding = base64.RawStdEncoding

// HashPassword returns a PHC-style Argon2id password hash.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("%w: %w", ErrRandomSaltFailure, err)
	}
	p := params{
		memory:      memoryKiB,
		time:        timeCost,
		parallelism: parallelism,
	}
	derivedKey := argon2.IDKey([]byte(password), salt, p.time, p.memory, p.parallelism, keyLength)
	return encode(p, salt, derivedKey), nil
}

// VerifyPassword checks password against a PHC-style Argon2id password hash.
func VerifyPassword(password string, encodedHash string) (bool, error) {
	if password == "" {
		return false, ErrEmptyPassword
	}
	p, salt, expectedKey, err := parse(encodedHash)
	if err != nil {
		return false, err
	}
	actualKey := argon2.IDKey([]byte(password), salt, p.time, p.memory, p.parallelism, uint32(len(expectedKey)))
	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1, nil
}

func encode(p params, salt []byte, derivedKey []byte) string {
	return fmt.Sprintf(
		"$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		algorithm,
		argon2.Version,
		p.memory,
		p.time,
		p.parallelism,
		encoding.EncodeToString(salt),
		encoding.EncodeToString(derivedKey),
	)
}

func parse(encodedHash string) (params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" {
		return params{}, nil, nil, ErrMalformedHash
	}
	if parts[1] != algorithm {
		return params{}, nil, nil, ErrUnsupportedHash
	}
	version, ok := strings.CutPrefix(parts[2], "v=")
	if !ok {
		return params{}, nil, nil, ErrMalformedHash
	}
	parsedVersion, err := strconv.Atoi(version)
	if err != nil {
		return params{}, nil, nil, ErrMalformedHash
	}
	if parsedVersion != argon2.Version {
		return params{}, nil, nil, ErrUnsupportedHash
	}
	p, err := parseParams(parts[3])
	if err != nil {
		return params{}, nil, nil, err
	}
	if p.memory != memoryKiB || p.time != timeCost || p.parallelism != parallelism {
		return params{}, nil, nil, ErrUnsupportedHash
	}
	salt, err := decodeRequired(parts[4])
	if err != nil {
		return params{}, nil, nil, err
	}
	if len(salt) != saltLength {
		return params{}, nil, nil, ErrUnsupportedHash
	}
	derivedKey, err := decodeRequired(parts[5])
	if err != nil {
		return params{}, nil, nil, err
	}
	if len(derivedKey) != keyLength {
		return params{}, nil, nil, ErrUnsupportedHash
	}
	return p, salt, derivedKey, nil
}

func parseParams(encoded string) (params, error) {
	values := strings.Split(encoded, ",")
	if len(values) != 3 {
		return params{}, ErrMalformedHash
	}
	parsed := make(map[string]uint64, 3)
	for _, value := range values {
		key, raw, ok := strings.Cut(value, "=")
		if !ok || raw == "" {
			return params{}, ErrMalformedHash
		}
		number, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || number == 0 {
			return params{}, ErrMalformedHash
		}
		parsed[key] = number
	}
	memory, ok := parsed["m"]
	if !ok {
		return params{}, ErrMalformedHash
	}
	time, ok := parsed["t"]
	if !ok {
		return params{}, ErrMalformedHash
	}
	lanes, ok := parsed["p"]
	if !ok || lanes > 255 {
		return params{}, ErrMalformedHash
	}
	return params{
		memory:      uint32(memory),
		time:        uint32(time),
		parallelism: uint8(lanes),
	}, nil
}

func decodeRequired(value string) ([]byte, error) {
	if value == "" {
		return nil, ErrMalformedHash
	}
	decoded, err := encoding.DecodeString(value)
	if err != nil {
		return nil, ErrMalformedHash
	}
	if len(decoded) == 0 {
		return nil, ErrMalformedHash
	}
	return decoded, nil
}
