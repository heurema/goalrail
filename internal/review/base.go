package review

import (
	"fmt"
	"strings"
)

// BaseResolution is the human-readable ref selected for a review and the
// immutable commit that ref named before any paid or mutating work began.
type BaseResolution struct {
	Ref    string
	Commit string
}

// ResolveBase resolves an explicit base or discovers one repository default
// from local remote-HEAD metadata. It deliberately performs no fetch and asks
// no hosting service: stale local metadata remains visible in the receipt, and
// ambiguous local metadata requires an explicit --base instead of a guess.
func ResolveBase(repositoryRoot, explicit string) (BaseResolution, error) {
	if explicit != "" {
		if strings.TrimSpace(explicit) == "" {
			return BaseResolution{}, fmt.Errorf("--base must name a non-empty Git ref")
		}
		commit, err := Resolve(repositoryRoot, explicit)
		if err != nil {
			return BaseResolution{}, fmt.Errorf("resolve explicit --base %q: %w", explicit, err)
		}
		return BaseResolution{Ref: explicit, Commit: commit}, nil
	}

	output, err := git(repositoryRoot,
		"for-each-ref", "--format=%(refname)%09%(symref)", "refs/remotes")
	if err != nil {
		return BaseResolution{}, fmt.Errorf("inspect local remote defaults: %w", err)
	}

	var candidates []BaseResolution
	for _, line := range strings.Split(output, "\n") {
		ref, target, found := strings.Cut(line, "\t")
		if !found || !strings.HasSuffix(ref, "/HEAD") ||
			!strings.HasPrefix(target, "refs/remotes/") || strings.HasSuffix(target, "/HEAD") {
			continue
		}
		commit, resolveErr := Resolve(repositoryRoot, target)
		if resolveErr != nil {
			continue
		}
		candidates = append(candidates, BaseResolution{
			Ref:    strings.TrimPrefix(target, "refs/remotes/"),
			Commit: commit,
		})
	}

	if len(candidates) != 1 {
		return BaseResolution{}, fmt.Errorf(
			"cannot resolve one unambiguous default base from local Git metadata (found %d); pass --base <ref> explicitly",
			len(candidates),
		)
	}
	return candidates[0], nil
}
