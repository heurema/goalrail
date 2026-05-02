package spine

import (
	"testing"

	"github.com/google/uuid"
)

func TestProjectContextIDGeneratorsReturnUUIDv7(t *testing.T) {
	tests := []struct {
		name     string
		generate func() (string, error)
	}{
		{name: "user", generate: func() (string, error) {
			id, err := NewUserID()
			return string(id), err
		}},
		{name: "installation", generate: func() (string, error) {
			id, err := NewInstallationID()
			return string(id), err
		}},
		{name: "organization", generate: func() (string, error) {
			id, err := NewOrganizationID()
			return string(id), err
		}},
		{name: "organization membership", generate: func() (string, error) {
			id, err := NewOrganizationMembershipID()
			return string(id), err
		}},
		{name: "project", generate: func() (string, error) {
			id, err := NewProjectID()
			return string(id), err
		}},
		{name: "repo binding", generate: func() (string, error) {
			id, err := NewRepoBindingID()
			return string(id), err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := tt.generate()
			if err != nil {
				t.Fatalf("generate() error = %v", err)
			}

			id, err := uuid.Parse(value)
			if err != nil {
				t.Fatalf("parse generated UUID: %v", err)
			}
			if id.Version() != 7 {
				t.Fatalf("generated UUID version = %d, want 7", id.Version())
			}
		})
	}
}
