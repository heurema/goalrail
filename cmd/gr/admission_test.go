package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestVerifyLineageCommandPublishesExactReadOnlyContract(t *testing.T) {
	var output bytes.Buffer
	if err := runVerifyLineage(context.Background(), []string{"--help"}, &output, &output); err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"-repo", "-base", "-head", "-packet", "-json"} {
		if !strings.Contains(output.String(), flag) {
			t.Fatalf("verify-lineage help omits %s: %q", flag, output.String())
		}
	}
	if !isCommand("verify-lineage") {
		t.Fatal("verify-lineage is not routed as a public command")
	}
}

func TestVerifyLineageUsageHasOperationalExitCode(t *testing.T) {
	err := runVerifyLineage(context.Background(), nil, &bytes.Buffer{}, &bytes.Buffer{})
	var coded interface{ ExitCode() int }
	if !errors.As(err, &coded) || coded.ExitCode() != 2 {
		t.Fatalf("usage error = %v, want exit 2", err)
	}
}
