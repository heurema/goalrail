package project

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/heurema/goalrail/internal/domain"
)

const (
	managedBlockStartPrefix  = "<!-- goalrail:managed-start schema=goalrail.agent-bootstrap/v1 digest="
	managedBlockEnd          = "<!-- goalrail:managed-end -->"
	MaxOwnerInstructionBytes = 128 << 10
)

type ManagedBlockAction string

const (
	ManagedBlockCreated   ManagedBlockAction = "created"
	ManagedBlockUpdated   ManagedBlockAction = "updated"
	ManagedBlockUnchanged ManagedBlockAction = "unchanged"
	ManagedBlockRefused   ManagedBlockAction = "refused"
)

type ManagedBlockPlan struct {
	Path    string             `json:"path"`
	Action  ManagedBlockAction `json:"action"`
	Reason  string             `json:"reason,omitempty"`
	Content []byte             `json:"-"`
}

func RenderManagedBlock(body []byte) []byte {
	body = bytes.TrimSpace(body)
	digest := domain.DigestCanonicalJSON(body)
	return []byte(fmt.Sprintf("%s%s -->\n%s\n%s\n", managedBlockStartPrefix, digest, body, managedBlockEnd))
}

// PlanManagedBlock returns a complete-file replacement while preserving every
// byte outside a uniquely delimited, known block. Unknown or ambiguous owner
// content is never rewritten.
func PlanManagedBlock(path string, current []byte, exists bool, desired []byte, knownPrevious ...[]byte) ManagedBlockPlan {
	return planManagedBlock(path, current, exists, desired, false, knownPrevious...)
}

// PlanManagedBlockDiscard applies the update command's explicit authority to a
// single, uniquely delimited Goalrail block. It still refuses missing,
// duplicated, nested, or malformed boundaries: discard authority is not
// permission to guess where owner bytes end.
func PlanManagedBlockDiscard(path string, current []byte, exists bool, desired []byte, knownPrevious ...[]byte) ManagedBlockPlan {
	return planManagedBlock(path, current, exists, desired, true, knownPrevious...)
}

func planManagedBlock(path string, current []byte, exists bool, desired []byte, discard bool, knownPrevious ...[]byte) ManagedBlockPlan {
	if !exists {
		return ManagedBlockPlan{Path: path, Action: ManagedBlockCreated, Content: append([]byte(nil), desired...)}
	}
	if len(current) > MaxOwnerInstructionBytes {
		return ManagedBlockPlan{Path: path, Action: ManagedBlockRefused, Reason: "owner instruction file exceeds the managed-block bound"}
	}
	startCount := bytes.Count(current, []byte(managedBlockStartPrefix))
	endCount := bytes.Count(current, []byte(managedBlockEnd))
	if startCount == 0 && endCount == 0 {
		return ManagedBlockPlan{Path: path, Action: ManagedBlockRefused, Reason: "owner instruction file has no Goalrail managed block"}
	}
	if startCount != 1 || endCount != 1 {
		return ManagedBlockPlan{Path: path, Action: ManagedBlockRefused, Reason: "managed block markers are duplicate or malformed"}
	}
	start := bytes.Index(current, []byte(managedBlockStartPrefix))
	lineEndOffset := bytes.IndexByte(current[start:], '\n')
	end := bytes.Index(current, []byte(managedBlockEnd))
	if lineEndOffset < 0 || end < start || bytes.Contains(current[start+len(managedBlockStartPrefix):start+lineEndOffset], []byte("<!--")) {
		return ManagedBlockPlan{Path: path, Action: ManagedBlockRefused, Reason: "managed block markers are nested or malformed"}
	}
	end += len(managedBlockEnd)
	if end < len(current) && current[end] == '\n' {
		end++
	}
	existing := current[start:end]
	if bytes.Equal(existing, desired) {
		return ManagedBlockPlan{Path: path, Action: ManagedBlockUnchanged, Content: append([]byte(nil), current...)}
	}
	known := false
	for _, previous := range knownPrevious {
		if bytes.Equal(existing, previous) {
			known = true
			break
		}
	}
	if !known && !discard {
		return ManagedBlockPlan{Path: path, Action: ManagedBlockRefused, Reason: "managed block differs from every retained canon"}
	}
	updated := make([]byte, 0, len(current)-len(existing)+len(desired))
	updated = append(updated, current[:start]...)
	updated = append(updated, desired...)
	updated = append(updated, current[end:]...)
	if strings.Count(string(updated), managedBlockStartPrefix) != 1 {
		return ManagedBlockPlan{Path: path, Action: ManagedBlockRefused, Reason: "managed block replacement would be ambiguous"}
	}
	return ManagedBlockPlan{Path: path, Action: ManagedBlockUpdated, Content: updated}
}
