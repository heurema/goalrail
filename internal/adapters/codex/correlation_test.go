package codex

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
)

const (
	testRepoRoot  = "/workspace/goalrail"
	testSessionID = "01900000-0000-7000-8000-000000000001"
)

func TestRunContextRoundTripsAndRejectsMutableFields(t *testing.T) {
	context := testRunContext(t, "intent-canary-v0", "run-001")
	encoded := environmentValue(t, context)
	decoded, err := DecodeRunContext([]byte(encoded))
	if err != nil {
		t.Fatalf("decode run context: %v", err)
	}
	if decoded.ChangeID() != context.ChangeID() || decoded.RunID() != context.RunID() || decoded.RepoRoot() != context.RepoRoot() {
		t.Fatalf("decoded context changed: %#v", decoded)
	}
	if _, err := DecodeRunContext([]byte(strings.TrimSuffix(encoded, "}") + `,"current_change":"other"}`)); err == nil {
		t.Fatal("run context accepted an undeclared mutable field")
	}
	if _, err := NewRunContext("change", "run", "/"); !errors.Is(err, ErrInvalidExpectedContext) {
		t.Fatalf("filesystem root boundary error = %v", err)
	}
}

func TestLifecycleHookPreservesRootAcrossStartupResumeAndCompaction(t *testing.T) {
	context := testRunContext(t, "intent-canary-v0", "run-001")
	encoded := environmentValue(t, context)
	var previous *CorrelationResult
	for _, source := range []string{"startup", "resume", "compact"} {
		result, err := BindLifecycleHook(context, encoded, sessionStartHook(t, testSessionID, source, nil), previous)
		if err != nil {
			t.Fatalf("bind %s hook: %v", source, err)
		}
		assertVerified(t, result, context, testSessionID, domain.SessionIdentityLifecycleHook)
		if result.ProviderEvent != "SessionStart" || result.ProviderMatcher != source {
			t.Fatalf("unexpected provider metadata: %#v", result)
		}
		current := result
		previous = &current
	}
}

func TestSubagentUsesProviderSuppliedParentSession(t *testing.T) {
	context := testRunContext(t, "intent-canary-v0", "run-001")
	encoded := environmentValue(t, context)
	root, err := BindLifecycleHook(context, encoded, sessionStartHook(t, testSessionID, "startup", nil), nil)
	if err != nil {
		t.Fatalf("bind root: %v", err)
	}
	extra := map[string]any{
		"turn_id":    "turn-001",
		"agent_id":   "agent-001",
		"agent_type": "worker",
	}
	child, err := BindLifecycleHook(context, encoded, hookJSON(t, testSessionID, "SubagentStart", extra), &root)
	if err != nil {
		t.Fatalf("bind child: %v", err)
	}
	assertVerified(t, child, context, testSessionID, domain.SessionIdentityLifecycleHook)
	if child.ProviderMatcher != "worker" {
		t.Fatalf("subagent matcher = %q", child.ProviderMatcher)
	}
}

func TestLifecycleHookReturnsExplicitUnlinkedReasons(t *testing.T) {
	context := testRunContext(t, "intent-canary-v0", "run-001")
	encoded := environmentValue(t, context)
	tests := []struct {
		name       string
		context    string
		hook       []byte
		previous   *CorrelationResult
		reasonCode domain.EvidenceReasonCode
	}{
		{
			name:       "missing run context",
			context:    "",
			hook:       sessionStartHook(t, testSessionID, "startup", nil),
			reasonCode: ReasonMissingRunContext,
		},
		{
			name:       "invalid run context",
			context:    `{not-json}`,
			hook:       sessionStartHook(t, testSessionID, "startup", nil),
			reasonCode: ReasonInvalidRunContext,
		},
		{
			name:    "conflicting repository",
			context: encoded,
			hook: sessionStartHook(t, testSessionID, "startup", map[string]any{
				"cwd": "/workspace/another-project",
			}),
			reasonCode: ReasonContextConflict,
		},
		{
			name:       "unsupported event",
			context:    encoded,
			hook:       hookJSON(t, testSessionID, "UserPromptSubmit", nil),
			reasonCode: ReasonUnsupportedHookEvent,
		},
		{
			name:       "invalid hook input",
			context:    encoded,
			hook:       []byte(`{"session_id":1}`),
			reasonCode: ReasonInvalidHookInput,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := BindLifecycleHook(context, test.context, test.hook, test.previous)
			if err != nil {
				t.Fatalf("bind hook: %v", err)
			}
			assertUnlinked(t, result, context, test.reasonCode)
		})
	}
}

func TestLifecycleHookDetectsSessionAndContextConflicts(t *testing.T) {
	context := testRunContext(t, "intent-canary-v0", "run-001")
	encoded := environmentValue(t, context)
	otherContext := testRunContext(t, "other-change", "run-002")
	crossLink, err := BindLifecycleHook(
		context,
		environmentValue(t, otherContext),
		sessionStartHook(t, testSessionID, "startup", nil),
		nil,
	)
	if err != nil {
		t.Fatalf("bind cross-run context: %v", err)
	}
	assertUnlinked(t, crossLink, context, ReasonContextConflict)

	initial, err := BindLifecycleHook(context, encoded, sessionStartHook(t, testSessionID, "startup", nil), nil)
	if err != nil {
		t.Fatalf("bind initial: %v", err)
	}

	changedSession, err := BindLifecycleHook(
		context,
		encoded,
		sessionStartHook(t, "01900000-0000-7000-8000-000000000002", "resume", nil),
		&initial,
	)
	if err != nil {
		t.Fatalf("bind changed session: %v", err)
	}
	assertUnlinked(t, changedSession, context, ReasonSessionConflict)
	if changedSession.ResolutionAttempts != 1 {
		t.Fatalf("session conflict attempts = %d, want terminal attempt 1", changedSession.ResolutionAttempts)
	}
	exhausted, err := BindLifecycleHook(
		context,
		encoded,
		sessionStartHook(t, testSessionID, "resume", nil),
		&changedSession,
	)
	if err != nil {
		t.Fatalf("retry after session conflict: %v", err)
	}
	assertUnlinked(t, exhausted, context, ReasonResolutionExhausted)

	conflictingPrevious := initial
	conflictingPrevious.Lineage.ChangeID = otherContext.ChangeID()
	conflict, err := BindLifecycleHook(context, encoded, sessionStartHook(t, testSessionID, "resume", nil), &conflictingPrevious)
	if err != nil {
		t.Fatalf("bind conflicting previous: %v", err)
	}
	assertUnlinked(t, conflict, context, ReasonContextConflict)
}

func TestOneResolutionAttemptCanResolvePriorUnlinkedResult(t *testing.T) {
	if MaxResolutionAttempts != 1 {
		t.Fatalf("resolution ceiling = %d, want 1", MaxResolutionAttempts)
	}
	context := testRunContext(t, "intent-canary-v0", "run-001")
	missing, err := BindLifecycleHook(context, "", sessionStartHook(t, testSessionID, "startup", nil), nil)
	if err != nil {
		t.Fatalf("bind missing context: %v", err)
	}
	resolved, err := BindLifecycleHook(
		context,
		environmentValue(t, context),
		sessionStartHook(t, testSessionID, "startup", nil),
		&missing,
	)
	if err != nil {
		t.Fatalf("resolve unlinked context: %v", err)
	}
	assertVerified(t, resolved, context, testSessionID, domain.SessionIdentityLifecycleHook)
	if resolved.ResolutionAttempts != 1 {
		t.Fatalf("resolution attempts = %d, want 1", resolved.ResolutionAttempts)
	}

	failedAgain, err := BindLifecycleHook(context, "", sessionStartHook(t, testSessionID, "startup", nil), &missing)
	if err != nil {
		t.Fatalf("run bounded resolution attempt: %v", err)
	}
	if failedAgain.ResolutionAttempts != 1 {
		t.Fatalf("failed resolution attempts = %d, want 1", failedAgain.ResolutionAttempts)
	}
	exhausted, err := BindLifecycleHook(context, environmentValue(t, context), sessionStartHook(t, testSessionID, "startup", nil), &failedAgain)
	if err != nil {
		t.Fatalf("evaluate exhausted resolution: %v", err)
	}
	assertUnlinked(t, exhausted, context, ReasonResolutionExhausted)
}

func TestLaunchReceiptUsesOnlyAuthoritativeThreadID(t *testing.T) {
	context := testRunContext(t, "intent-canary-v0", "run-001")
	valid := []byte(`{"threadId":"01900000-0000-7000-8000-000000000010","title":"ignored"}`)
	result, err := BindLaunchReceipt(context, valid, nil)
	if err != nil {
		t.Fatalf("bind launch receipt: %v", err)
	}
	assertVerified(t, result, context, "01900000-0000-7000-8000-000000000010", domain.SessionIdentityLaunchReceipt)
	if result.ProviderEvent != "create_thread" || result.ProviderMatcher != "threadId" {
		t.Fatalf("unexpected launch metadata: %#v", result)
	}

	for name, receipt := range map[string][]byte{
		"manual task has no receipt":            nil,
		"queued client ID is not root identity": []byte(`{"clientThreadId":"client-001"}`),
	} {
		t.Run(name, func(t *testing.T) {
			unlinkedResult, err := BindLaunchReceipt(context, receipt, nil)
			if err != nil {
				t.Fatalf("bind absent receipt: %v", err)
			}
			assertUnlinked(t, unlinkedResult, context, ReasonMissingLaunchReceipt)
		})
	}

	invalid, err := BindLaunchReceipt(context, []byte(`{"threadId":"raw identity with spaces"}`), nil)
	if err != nil {
		t.Fatalf("bind invalid receipt: %v", err)
	}
	assertUnlinked(t, invalid, context, ReasonInvalidLaunchReceipt)
}

func TestLaunchReceiptCannotReplaceExistingRootOrIdentitySource(t *testing.T) {
	context := testRunContext(t, "intent-canary-v0", "run-001")
	first, err := BindLaunchReceipt(context, []byte(`{"threadId":"thread-001"}`), nil)
	if err != nil {
		t.Fatalf("bind first launch receipt: %v", err)
	}
	changed, err := BindLaunchReceipt(context, []byte(`{"threadId":"thread-002"}`), &first)
	if err != nil {
		t.Fatalf("bind changed launch receipt: %v", err)
	}
	assertUnlinked(t, changed, context, ReasonSessionConflict)

	hook, err := BindLifecycleHook(
		context,
		environmentValue(t, context),
		sessionStartHook(t, "thread-001", "startup", nil),
		&first,
	)
	if err != nil {
		t.Fatalf("bind conflicting identity source: %v", err)
	}
	assertUnlinked(t, hook, context, ReasonIdentitySourceConflict)
}

func TestConcurrentRunContextsNeverCrossLink(t *testing.T) {
	contexts := []RunContext{
		testRunContext(t, "change-one", "run-one"),
		testRunContext(t, "change-two", "run-two"),
	}
	sessions := []string{
		"01900000-0000-7000-8000-000000000001",
		"01900000-0000-7000-8000-000000000002",
	}
	results := make([]CorrelationResult, len(contexts))
	errorsSeen := make([]error, len(contexts))
	var wait sync.WaitGroup
	for index := range contexts {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			results[index], errorsSeen[index] = BindLifecycleHook(
				contexts[index],
				environmentValue(t, contexts[index]),
				sessionStartHook(t, sessions[index], "startup", nil),
				nil,
			)
		}(index)
	}
	wait.Wait()
	for index := range results {
		if errorsSeen[index] != nil {
			t.Fatalf("bind concurrent context %d: %v", index, errorsSeen[index])
		}
		assertVerified(t, results[index], contexts[index], sessions[index], domain.SessionIdentityLifecycleHook)
	}
	if results[0].Lineage.ContextDigest == results[1].Lineage.ContextDigest {
		t.Fatal("concurrent immutable contexts produced the same digest")
	}
}

func TestHookPayloadAndTranscriptFieldsAreNotRetained(t *testing.T) {
	context := testRunContext(t, "intent-canary-v0", "run-001")
	extra := map[string]any{
		"transcript_path": "/private/raw-transcript.jsonl",
		"prompt":          "do not retain this secret prompt",
	}
	result, err := BindLifecycleHook(
		context,
		environmentValue(t, context),
		sessionStartHook(t, testSessionID, "startup", extra),
		nil,
	)
	if err != nil {
		t.Fatalf("bind hook: %v", err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("encode result: %v", err)
	}
	if strings.Contains(string(encoded), "transcript") || strings.Contains(string(encoded), "secret prompt") {
		t.Fatalf("correlation result retained raw provider payload: %s", encoded)
	}
}

func testRunContext(t *testing.T, changeID, runID string) RunContext {
	t.Helper()
	context, err := NewRunContext(domain.ChangeID(changeID), domain.RunID(runID), testRepoRoot)
	if err != nil {
		t.Fatalf("create run context: %v", err)
	}
	return context
}

func environmentValue(t *testing.T, context RunContext) string {
	t.Helper()
	encoded, err := context.EnvironmentValue()
	if err != nil {
		t.Fatalf("encode run context: %v", err)
	}
	return encoded
}

func sessionStartHook(t *testing.T, sessionID, source string, overrides map[string]any) []byte {
	t.Helper()
	fields := map[string]any{"source": source}
	for key, value := range overrides {
		fields[key] = value
	}
	return hookJSON(t, sessionID, "SessionStart", fields)
}

func hookJSON(t *testing.T, sessionID, event string, extra map[string]any) []byte {
	t.Helper()
	payload := map[string]any{
		"session_id":      sessionID,
		"cwd":             testRepoRoot,
		"hook_event_name": event,
	}
	for key, value := range extra {
		payload[key] = value
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode hook: %v", err)
	}
	return encoded
}

func assertVerified(
	t *testing.T,
	result CorrelationResult,
	context RunContext,
	sessionID string,
	source domain.SessionIdentitySource,
) {
	t.Helper()
	if !result.Verified() {
		t.Fatalf("result is not verified: %#v", result)
	}
	if result.Lineage.ChangeID != context.ChangeID() ||
		result.Lineage.RunID != context.RunID() ||
		result.Lineage.RootSessionID != domain.SessionID(sessionID) ||
		result.Lineage.IdentitySource != source ||
		result.Lineage.UnlinkedReasonCode != "" {
		t.Fatalf("unexpected verified lineage: %#v", result.Lineage)
	}
}

func assertUnlinked(t *testing.T, result CorrelationResult, context RunContext, reason domain.EvidenceReasonCode) {
	t.Helper()
	if result.Verified() || result.Lineage.Status != domain.LineageUnlinked {
		t.Fatalf("result is not unlinked: %#v", result)
	}
	if result.Lineage.ChangeID != context.ChangeID() ||
		result.Lineage.RunID != context.RunID() ||
		result.Lineage.RootSessionID != "" ||
		result.Lineage.IdentitySource != "" ||
		result.Lineage.UnlinkedReasonCode != reason {
		t.Fatalf("unexpected unlinked lineage: %#v", result.Lineage)
	}
	if result.ProviderEvent != "" || result.ProviderMatcher != "" {
		t.Fatalf("unlinked result claimed provider metadata: %#v", result)
	}
}

func TestZeroRunContextFailsAsProgrammingError(t *testing.T) {
	_, err := BindLifecycleHook(RunContext{}, "", nil, nil)
	if !errors.Is(err, ErrInvalidExpectedContext) {
		t.Fatalf("zero run context error = %v", err)
	}
	if !strings.Contains(fmt.Sprint(err), "change and run IDs") {
		t.Fatalf("zero context error is not actionable: %v", err)
	}
}
