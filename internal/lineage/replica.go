package lineage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/localrun"
)

const MaxReplicaBytes = 1 << 20

type Replica struct {
	Digest    domain.SHA256Digest
	Schema    string
	Reference string
	Canonical []byte
}

func PrepareReplica(reader io.Reader, expectedDigest domain.SHA256Digest, expectedSchema string) (Replica, error) {
	if !domain.IsSHA256Digest(expectedDigest) {
		return Replica{}, fmt.Errorf("replica expected digest must be a complete SHA-256 reference")
	}
	if !validReplicaSchema(expectedSchema) {
		return Replica{}, fmt.Errorf("replica expected schema must be one bounded versioned identifier")
	}
	raw, err := io.ReadAll(io.LimitReader(reader, MaxReplicaBytes+1))
	if err != nil {
		return Replica{}, fmt.Errorf("read evidence replica: %w", err)
	}
	if len(raw) > MaxReplicaBytes {
		return Replica{}, fmt.Errorf("evidence replica exceeds %d bytes", MaxReplicaBytes)
	}
	if domain.DigestCanonicalJSON(raw) != expectedDigest {
		return Replica{}, fmt.Errorf("evidence replica digest mismatch")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return Replica{}, fmt.Errorf("decode evidence replica: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Replica{}, fmt.Errorf("evidence replica contains trailing JSON")
	}
	object, ok := value.(map[string]any)
	if !ok {
		return Replica{}, fmt.Errorf("evidence replica must be one JSON object")
	}
	observedSchema, ok := object["schema"].(string)
	if !ok || observedSchema != expectedSchema {
		return Replica{}, fmt.Errorf("evidence replica schema mismatch: got %q, want %q", observedSchema, expectedSchema)
	}
	if err := validatePortablePayload(value, "$"); err != nil {
		return Replica{}, err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return Replica{}, fmt.Errorf("canonicalize evidence replica: %w", err)
	}
	known, err := validateKnownReplica(expectedSchema, raw)
	if err != nil {
		return Replica{}, err
	}
	if !known && !bytes.Equal(raw, canonical) {
		return Replica{}, fmt.Errorf("evidence replica is not exact canonical JSON")
	}
	return Replica{
		Digest: expectedDigest, Schema: expectedSchema,
		Reference: ".goalrail/evidence/sha256/" + digestComponent(expectedDigest),
		Canonical: append([]byte(nil), raw...),
	}, nil
}

func validateKnownReplica(schema string, raw []byte) (bool, error) {
	switch schema {
	case domain.WorkSpecSchemaV1:
		value, err := domain.DecodeWorkSpec(bytes.NewReader(raw))
		if err != nil {
			return true, fmt.Errorf("validate WorkSpec replica: %w", err)
		}
		artifact, err := domain.FreezeWorkSpec(value)
		if err != nil || !bytes.Equal(raw, artifact.CanonicalJSON()) {
			return true, fmt.Errorf("WorkSpec replica is not the canonical v1 artifact")
		}
	case domain.WorkUnitSchemaV1:
		value, err := domain.DecodeWorkUnit(bytes.NewReader(raw))
		if err != nil {
			return true, fmt.Errorf("validate work-unit replica: %w", err)
		}
		artifact, err := domain.FreezeWorkUnit(value)
		if err != nil || !bytes.Equal(raw, artifact.CanonicalJSON()) {
			return true, fmt.Errorf("work-unit replica is not the canonical v1 artifact")
		}
	case domain.LineageEventSchemaV1:
		value, err := domain.DecodeLineageEvent(bytes.NewReader(raw))
		if err != nil {
			return true, fmt.Errorf("validate lineage-event replica: %w", err)
		}
		artifact, err := domain.FreezeLineageEvent(value)
		if err != nil || !bytes.Equal(raw, artifact.CanonicalJSON()) {
			return true, fmt.Errorf("lineage-event replica is not the canonical v1 artifact")
		}
	case localrun.TerminalReceiptSchemaV1:
		value, err := localrun.DecodeTerminalReceipt(bytes.NewReader(raw))
		if err != nil {
			return true, fmt.Errorf("validate terminal-receipt replica: %w", err)
		}
		canonical, err := localrun.CanonicalTerminalReceipt(value)
		if err != nil || !bytes.Equal(raw, canonical) {
			return true, fmt.Errorf("terminal-receipt replica is not the canonical v1 artifact")
		}
	default:
		return false, nil
	}
	return true, nil
}

func validReplicaSchema(value string) bool {
	if value == "" || len(value) > 128 || !strings.Contains(value, "/") || domain.ContainsSecretShapedContent(value) {
		return false
	}
	for _, character := range value {
		if character <= ' ' || character > '~' {
			return false
		}
	}
	return true
}

func validatePortablePayload(value any, path string) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			normalized := strings.ToLower(strings.TrimSpace(key))
			switch normalized {
			case "raw", "raw_content", "body", "content", "prompt", "transcript", "comment_body", "review_body",
				"source_body", "request", "response", "credential", "credentials", "token", "secret":
				return fmt.Errorf("evidence replica contains prohibited payload field %s.%s", path, key)
			}
			if err := validatePortablePayload(nested, path+"."+key); err != nil {
				return err
			}
		}
	case []any:
		for index, nested := range typed {
			if err := validatePortablePayload(nested, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
		}
	case string:
		if domain.ContainsSecretShapedContent(typed) {
			return fmt.Errorf("evidence replica contains secret-shaped content at %s", path)
		}
	}
	return nil
}
