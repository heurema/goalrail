package githubadmission

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/heurema/goalrail/internal/domain"
)

const CheckoutActionCommit = "11bd71901bbe5b1630ceea73d27597364c9af683"

type ReleasePin struct {
	Version     string
	ArchiveURL  string
	ArchiveName string
	SHA256      domain.SHA256Digest
}

// RenderWorkflow refuses floating releases. The returned workflow is prepared
// repository content only; it never enables a required check or ruleset.
func RenderWorkflow(pin ReleasePin) ([]byte, error) {
	parsed, err := url.Parse(pin.ArchiveURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("Goalrail workflow release URL must be immutable credential-free HTTPS")
	}
	if !domain.IsCanonicalID(pin.Version) || !domain.IsSHA256Digest(pin.SHA256) || path.Base(parsed.Path) != pin.ArchiveName || path.Base(pin.ArchiveName) != pin.ArchiveName {
		return nil, errors.New("Goalrail workflow requires an exact version, archive name, and SHA-256 pin")
	}
	for _, value := range []string{pin.Version, pin.ArchiveURL, pin.ArchiveName, string(pin.SHA256)} {
		if strings.ContainsAny(value, "'\r\n") || domain.ContainsSecretShapedContent(value) {
			return nil, errors.New("Goalrail workflow pin contains unsafe content")
		}
	}
	digest := strings.TrimPrefix(string(pin.SHA256), "sha256:")
	workflow := fmt.Sprintf(`name: Goalrail lineage

on:
  pull_request:
    types: [opened, reopened, synchronize, ready_for_review, closed]
  pull_request_review:
    types: [submitted, edited, dismissed]

permissions:
  contents: read
  pull-requests: read
  checks: read

concurrency:
  group: goalrail-lineage-${{ github.event.pull_request.number }}-${{ github.event.pull_request.head.sha }}
  cancel-in-progress: true

jobs:
  lineage:
    name: goalrail/lineage
    runs-on: ubuntu-24.04
    steps:
      - name: Check out the frozen pull-request head
        uses: actions/checkout@%s
        with:
          ref: ${{ github.event.pull_request.head.sha }}
          fetch-depth: 0
          persist-credentials: false
      - name: Verify pinned Goalrail release artifact
        shell: bash
        env:
          GOALRAIL_ARCHIVE_URL: '%s'
          GOALRAIL_ARCHIVE_NAME: '%s'
          GOALRAIL_ARCHIVE_SHA256: '%s'
          GOALRAIL_VERSION: '%s'
        run: |
          set -euo pipefail
          install -d -m 0700 "${RUNNER_TEMP}/goalrail-bin"
          curl --fail --location --proto '=https' --tlsv1.2 \
            --output "${RUNNER_TEMP}/${GOALRAIL_ARCHIVE_NAME}" \
            "${GOALRAIL_ARCHIVE_URL}"
          printf '%%s  %%s\n' "${GOALRAIL_ARCHIVE_SHA256}" \
            "${RUNNER_TEMP}/${GOALRAIL_ARCHIVE_NAME}" | sha256sum --check --strict
          tar -xzf "${RUNNER_TEMP}/${GOALRAIL_ARCHIVE_NAME}" \
            -C "${RUNNER_TEMP}/goalrail-bin"
          test -x "${RUNNER_TEMP}/goalrail-bin/gr"
          "${RUNNER_TEMP}/goalrail-bin/gr" version \
            | grep --fixed-strings --quiet "\"version\":\"${GOALRAIL_VERSION}\""
      - name: Collect bounded provider evidence and verify lineage
        shell: bash
        env:
          GITHUB_TOKEN: ${{ github.token }}
          GOALRAIL_EVENT_NAME: ${{ github.event_name }}
        run: |
          set -euo pipefail
          packet="${RUNNER_TEMP}/goalrail-admission-packet.json"
          trap 'rm -f "${packet}"' EXIT
          gr="${RUNNER_TEMP}/goalrail-bin/gr"
          base="${{ github.event.pull_request.base.sha }}"
          head="${{ github.event.pull_request.head.sha }}"
          "${gr}" github-collect \
            --repo . --base "${base}" --head "${head}" \
            --event-name "${GOALRAIL_EVENT_NAME}" \
            --event "${GITHUB_EVENT_PATH}" --output "${packet}"
          "${gr}" verify-lineage --repo . --base "${base}" --head "${head}" \
            --packet "${packet}" --json
`, CheckoutActionCommit, pin.ArchiveURL, pin.ArchiveName, digest, pin.Version)
	return []byte(workflow), nil
}
