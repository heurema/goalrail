package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// SHA256Digest is the lowercase, algorithm-qualified identity used by v1
// canonical artifacts.
type SHA256Digest string

// CanonicalArtifact retains deterministic JSON bytes and their content
// identity. It deliberately carries no mutable decoded value.
type CanonicalArtifact struct {
	canonical []byte
	digest    SHA256Digest
}

func newCanonicalArtifact(value any) (CanonicalArtifact, error) {
	canonical, err := json.Marshal(value)
	if err != nil {
		return CanonicalArtifact{}, fmt.Errorf("encode canonical artifact: %w", err)
	}
	return CanonicalArtifact{
		canonical: canonical,
		digest:    DigestCanonicalJSON(canonical),
	}, nil
}

// CanonicalJSON returns a copy of the frozen bytes.
func (artifact CanonicalArtifact) CanonicalJSON() []byte {
	return append([]byte(nil), artifact.canonical...)
}

// Digest returns the content identity of CanonicalJSON.
func (artifact CanonicalArtifact) Digest() SHA256Digest {
	return artifact.digest
}

// DigestCanonicalJSON returns a lowercase SHA-256 reference for content.
func DigestCanonicalJSON(content []byte) SHA256Digest {
	sum := sha256.Sum256(content)
	return SHA256Digest("sha256:" + hex.EncodeToString(sum[:]))
}

// IsSHA256Digest reports whether value is a complete lowercase SHA-256
// reference rather than a shortened or algorithm-implicit digest.
func IsSHA256Digest(value SHA256Digest) bool {
	return hexDigestPattern.MatchString(string(value))
}

func decodeStrictBoundedJSON[T any](reader io.Reader, maxBytes int, label string) (T, error) {
	var zero T
	raw, err := io.ReadAll(io.LimitReader(reader, int64(maxBytes)+1))
	if err != nil {
		return zero, fmt.Errorf("read %s: %w", label, err)
	}
	if len(raw) > maxBytes {
		return zero, fmt.Errorf("decode %s: payload exceeds %d bytes", label, maxBytes)
	}

	var value T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return zero, fmt.Errorf("decode %s: %w", label, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("unexpected trailing JSON value")
		}
		return zero, fmt.Errorf("decode %s: %w", label, err)
	}
	return value, nil
}

func normalizeStringSet(values []string) []string {
	normalized := append([]string(nil), values...)
	sort.Strings(normalized)
	return normalized
}

func duplicateStrings(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}
