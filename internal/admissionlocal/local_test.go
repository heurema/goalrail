package admissionlocal

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
)

func TestCommitMessageTrailerIsOnlyAcceptedFromTheFooter(t *testing.T) {
	valid := "feat(core): bind work\n\nWhy:\n- bounded change\n\nGoalrail-Work-Unit: wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"
	id, err := ValidateCommitMessage(strings.NewReader(valid))
	if err != nil || id != "wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("valid trailer = %q, %v", id, err)
	}
	for name, message := range map[string]string{
		"missing":    "feat: change\n",
		"body-only":  "Goalrail-Work-Unit: wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n\nnot a trailer\n",
		"invalid":    "feat: change\n\nGoalrail-Work-Unit: nope\n",
		"duplicated": "feat: change\n\nGoalrail-Work-Unit: wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\nGoalrail-Work-Unit: wu_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ValidateCommitMessage(strings.NewReader(message)); err == nil {
				t.Fatal("invalid trailer was accepted")
			}
		})
	}
	if _, err := ValidateCommitMessage(strings.NewReader("feat: change\n")); !errors.Is(err, ErrTrailerMissing) {
		t.Fatalf("missing trailer error = %v", err)
	}
}

func TestLocalShimsOnlyDelegateToPublicChecks(t *testing.T) {
	shims, err := RenderShims("/opt/goalrail/bin/gr")
	if err != nil {
		t.Fatal(err)
	}
	joined := bytes.Join([][]byte{shims.Doctor, shims.Verify, shims.CommitMsg}, nil)
	for _, required := range []string{" doctor ", " verify-lineage ", " commit-msg "} {
		if !bytes.Contains(joined, []byte(required)) {
			t.Fatalf("shim omitted delegation %q", required)
		}
	}
	for _, forbidden := range []string{"material", "exception", "VALID", "allow", "deny"} {
		if bytes.Contains(joined, []byte(forbidden)) {
			t.Fatalf("shim contains decision logic word %q", forbidden)
		}
	}
	if _, err := RenderShims("/tmp/gr;rm"); err == nil {
		t.Fatal("unsafe executable was accepted")
	}
}

func TestPrekIsGeneratedOnlyForTheExactSelectedProfile(t *testing.T) {
	profile := domain.SetupProfile{}
	if raw, selected, err := RenderPrekAdapter(profile, "/opt/goalrail/bin/gr"); err != nil || selected || raw != nil {
		t.Fatalf("unselected prek = %q, %v, %v", raw, selected, err)
	}
	profile.ScaffoldAdapters = append(profile.ScaffoldAdapters, domain.SetupAdapterPin{
		ID: PrekAdapterID, Version: PrekAdapterVersion, SourceRef: PrekAdapterSourceRef, Integrity: PrekTemplateDigest(),
	})
	raw, selected, err := RenderPrekAdapter(profile, "/opt/goalrail/bin/gr")
	if err != nil || !selected || !bytes.Contains(raw, []byte("Goalrail diagnosis (advisory)")) {
		t.Fatalf("selected prek = %q, %v, %v", raw, selected, err)
	}
	if bytes.Contains(bytes.ToLower(raw), []byte("install")) {
		t.Fatal("generated adapter attempts to install prek")
	}
	profile.ScaffoldAdapters[0].Integrity = domain.DigestCanonicalJSON([]byte("different"))
	if _, _, err := RenderPrekAdapter(profile, "/opt/goalrail/bin/gr"); err == nil {
		t.Fatal("mismatched prek pin was accepted")
	}
}
