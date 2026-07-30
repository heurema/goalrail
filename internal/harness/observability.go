package harness

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ObservabilityFile is where a bring-your-own endpoint may be configured, inside
// the state root rather than the repository. Credentials never enter repository
// content, so they never enter a commit, a diff, or a review.
const ObservabilityFile = "observability.json"

// Environment variables recognised for an already-running instance. They are read
// only to answer "is it configured"; no value beyond the endpoint is ever
// reported, and no key is ever echoed.
const (
	observabilityEndpointEnvironment  = "LANGFUSE_HOST"
	observabilityPublicKeyEnvironment = "LANGFUSE_PUBLIC_KEY"
	observabilitySecretKeyEnvironment = "LANGFUSE_SECRET_KEY"
)

// ObservabilityState is what a diagnosis says about the optional exporter.
//
// Absence is a fully working configuration: the local state root is the source of
// truth and an exporter is a removable attachment. So this carries no next action
// and never contributes a failure.
type ObservabilityState struct {
	Configured bool `json:"configured"`

	// Endpoint is the most a report may name. Keys are read to decide
	// `Configured` and are never returned, printed, or logged.
	Endpoint string `json:"endpoint,omitempty"`

	// Source says where the configuration was found, so a user who forgot which
	// of the two places they used does not have to guess.
	Source string `json:"source,omitempty"`

	// Note is the one line a report prints when nothing is configured.
	Note string `json:"note,omitempty"`
}

// InspectObservability reports whether an exporter is configured, reading the
// environment first and then the state root.
func InspectObservability(stateRoot string, lookupEnvironment func(string) string) ObservabilityState {
	if lookupEnvironment == nil {
		lookupEnvironment = os.Getenv
	}
	endpoint := strings.TrimSpace(lookupEnvironment(observabilityEndpointEnvironment))
	publicKey := strings.TrimSpace(lookupEnvironment(observabilityPublicKeyEnvironment))
	secretKey := strings.TrimSpace(lookupEnvironment(observabilitySecretKeyEnvironment))
	if endpoint != "" && publicKey != "" && secretKey != "" {
		return ObservabilityState{Configured: true, Endpoint: redactedEndpoint(endpoint), Source: "environment"}
	}

	if strings.TrimSpace(stateRoot) != "" {
		raw, err := os.ReadFile(filepath.Join(stateRoot, ObservabilityFile))
		if err == nil {
			var configured struct {
				Endpoint  string `json:"endpoint"`
				PublicKey string `json:"public_key"`
				SecretKey string `json:"secret_key"`
			}
			if json.Unmarshal(raw, &configured) == nil &&
				strings.TrimSpace(configured.Endpoint) != "" &&
				strings.TrimSpace(configured.PublicKey) != "" &&
				strings.TrimSpace(configured.SecretKey) != "" {
				return ObservabilityState{
					Configured: true,
					Endpoint:   redactedEndpoint(strings.TrimSpace(configured.Endpoint)),
					Source:     "state root",
				}
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			// An unreadable file is still not a failure of the harness; it is one
			// line saying observability could not be read.
			return ObservabilityState{Note: "observability: configuration could not be read (optional)"}
		}
	}
	return ObservabilityState{Note: "observability: not configured (optional)"}
}

// redactedEndpoint strips credentials a URL may embed before the endpoint is
// reported. "The endpoint is the most a report may name" has to survive a user
// who configured https://user:secret@host — reflecting that verbatim would print
// the secret in the one line that promises not to.
func redactedEndpoint(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.User == nil {
		return endpoint
	}
	parsed.User = nil
	return parsed.String()
}
