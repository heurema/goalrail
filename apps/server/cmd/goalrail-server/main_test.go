package main

import (
	"errors"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/bootstrapowner"
)

func TestParseBootstrapOwnerFlags(t *testing.T) {
	input, err := parseBootstrapOwnerFlags([]string{
		"--email", "Owner@Example.COM",
		"--display-name", "Owner User",
		"--organization-slug", "Primary",
		"--organization-name", "Primary Org",
		"--public-base-url", "https://goalrail.example.com/",
	})
	if err != nil {
		t.Fatalf("parseBootstrapOwnerFlags() error = %v", err)
	}
	if input.Email != "owner@example.com" {
		t.Fatalf("Email = %q, want normalized email", input.Email)
	}
	if input.OrganizationSlug != "primary" {
		t.Fatalf("OrganizationSlug = %q, want normalized slug", input.OrganizationSlug)
	}
	if input.PublicBaseURL != "https://goalrail.example.com" {
		t.Fatalf("PublicBaseURL = %q, want normalized URL", input.PublicBaseURL)
	}
}

func TestParseBootstrapOwnerFlagsRequiresInputs(t *testing.T) {
	_, err := parseBootstrapOwnerFlags([]string{
		"--email", "owner@example.com",
		"--display-name", "Owner User",
		"--organization-slug", "primary",
		"--organization-name", "Primary Org",
	})
	if !errors.Is(err, bootstrapowner.ErrInvalidInput) {
		t.Fatalf("parseBootstrapOwnerFlags() error = %v, want ErrInvalidInput", err)
	}
}
