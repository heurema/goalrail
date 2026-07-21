package domain

import "testing"

func TestAssessmentKeepsChecksIndependentFromIntentOutcome(t *testing.T) {
	assessment := Assessment{
		Outcome:     IntentMiss,
		ChecksGreen: true,
	}

	if !assessment.ChecksGreen || assessment.Outcome != IntentMiss {
		t.Fatal("green checks must not overwrite the owner intent outcome")
	}
}

func TestCanonicalSessionIdentitySourceConstants(t *testing.T) {
	sources := []SessionIdentitySource{
		SessionIdentityLifecycleHook,
		SessionIdentityLaunchReceipt,
	}

	if sources[0] != "lifecycle_hook" || sources[1] != "launch_receipt" {
		t.Fatalf("unexpected identity sources: %v", sources)
	}
}
