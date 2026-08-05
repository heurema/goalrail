package project

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
)

type ClaimState string
type ClaimReason string

const (
	ClaimUnmanaged       ClaimState = "unmanaged"
	ClaimManaged         ClaimState = "managed"
	ClaimDeclaredInvalid ClaimState = "declared_invalid"

	ReasonDeclarationUnreadable   ClaimReason = "DECLARATION_UNREADABLE"
	ReasonDeclarationUnsafePath   ClaimReason = "DECLARATION_UNSAFE_PATH"
	ReasonDeclarationNotRegular   ClaimReason = "DECLARATION_NOT_REGULAR"
	ReasonDeclarationTooLarge     ClaimReason = "DECLARATION_TOO_LARGE"
	ReasonDeclarationMalformed    ClaimReason = "DECLARATION_MALFORMED"
	ReasonDeclarationNonCanonical ClaimReason = "DECLARATION_NON_CANONICAL"
	ReasonDeclarationChanged      ClaimReason = "DECLARATION_CHANGED"
)

var (
	ErrClaimNotManaged = errors.New("project claim is not managed")
	ErrClaimInvalid    = errors.New("project declaration claim is invalid")
	ErrClaimChanged    = errors.New("project declaration changed after validation")
)

// Inspection is the fail-closed result of reading the reserved declaration
// claim. An absent path is unmanaged; every unsafe or invalid present path is
// declared-invalid.
type Inspection struct {
	State             ClaimState
	Reason            ClaimReason
	Detail            string
	WorktreeRoot      string
	DeclarationPath   string
	Declaration       domain.ProjectDeclaration
	DeclarationDigest domain.SHA256Digest
	snapshot          *claimSnapshot
}

type claimSnapshot struct {
	info   os.FileInfo
	raw    []byte
	digest domain.SHA256Digest
}

type regularFileReader func(path string, label string, limit int) ([]byte, os.FileInfo, error)

// Inspect resolves start to its worktree root and inspects only the canonical
// committed project declaration. It never consults an ambient marker.
func Inspect(ctx context.Context, start string) (Inspection, error) {
	return inspectWithReader(ctx, start, boundedio.ReadRegularFileWithInfo)
}

func inspectWithReader(ctx context.Context, start string, read regularFileReader) (Inspection, error) {
	root, err := ResolveWorktreeRoot(ctx, start)
	if err != nil {
		return Inspection{}, err
	}
	declarationPath := filepath.Join(root, filepath.FromSlash(domain.ProjectDeclarationPath))
	inspection := Inspection{
		State:           ClaimUnmanaged,
		WorktreeRoot:    root,
		DeclarationPath: declarationPath,
	}

	before, err := os.Lstat(declarationPath)
	if errors.Is(err, fs.ErrNotExist) {
		return inspection, nil
	}
	if err != nil {
		return invalidInspection(inspection, ReasonDeclarationUnreadable, err), nil
	}
	if err := ensureSafeRegularPath(root, declarationPath, before); err != nil {
		reason := ReasonDeclarationUnsafePath
		if before.Mode()&os.ModeSymlink == 0 && !before.Mode().IsRegular() {
			reason = ReasonDeclarationNotRegular
		}
		return invalidInspection(inspection, reason, err), nil
	}

	raw, opened, err := read(declarationPath, "project declaration", domain.MaxProjectDeclarationBytes)
	if err != nil {
		reason := ReasonDeclarationUnreadable
		if strings.Contains(err.Error(), "exceeds") {
			reason = ReasonDeclarationTooLarge
		}
		return invalidInspection(inspection, reason, err), nil
	}
	after, err := os.Lstat(declarationPath)
	if err == nil {
		err = ensureSafeRegularPath(root, declarationPath, after)
	}
	if err != nil || !sameFileSnapshot(before, opened) || !sameFileSnapshot(opened, after) {
		if err == nil {
			err = ErrClaimChanged
		}
		return invalidInspection(inspection, ReasonDeclarationChanged, err), nil
	}

	declaration, err := domain.DecodeProjectDeclaration(bytes.NewReader(raw))
	if err != nil {
		return invalidInspection(inspection, ReasonDeclarationMalformed, err), nil
	}
	frozen, err := domain.FreezeProjectDeclaration(declaration)
	if err != nil {
		return invalidInspection(inspection, ReasonDeclarationMalformed, err), nil
	}
	if !bytes.Equal(raw, frozen.CanonicalJSON()) {
		return invalidInspection(inspection, ReasonDeclarationNonCanonical, errors.New("project declaration bytes are not canonical JSON")), nil
	}
	for _, reference := range []domain.CommittedArtifactReference{
		declaration.Policy,
		declaration.Bootstrap,
		declaration.SetupProfile,
	} {
		if !pathContained(root, filepath.Join(root, filepath.FromSlash(reference.Path))) {
			return invalidInspection(inspection, ReasonDeclarationUnsafePath, fmt.Errorf("referenced path %q escapes the worktree", reference.Path)), nil
		}
	}

	inspection.State = ClaimManaged
	inspection.Declaration = declaration
	inspection.DeclarationDigest = frozen.Digest()
	inspection.snapshot = &claimSnapshot{
		info:   opened,
		raw:    append([]byte(nil), raw...),
		digest: frozen.Digest(),
	}
	return inspection, nil
}

// Revalidate checks the exact declaration identity immediately before a
// dependent mutation. Any byte, inode, mode, size, or timestamp change fails
// closed even when the replacement is otherwise valid.
func (inspection Inspection) Revalidate() error {
	return inspection.revalidateWithReader(boundedio.ReadRegularFileWithInfo)
}

func (inspection Inspection) revalidateWithReader(read regularFileReader) error {
	if inspection.State == ClaimUnmanaged {
		return ErrClaimNotManaged
	}
	if inspection.State != ClaimManaged || inspection.snapshot == nil {
		return ErrClaimInvalid
	}
	before, err := os.Lstat(inspection.DeclarationPath)
	if err != nil {
		return fmt.Errorf("%w: lstat declaration: %v", ErrClaimChanged, err)
	}
	if err := ensureSafeRegularPath(inspection.WorktreeRoot, inspection.DeclarationPath, before); err != nil {
		return fmt.Errorf("%w: %v", ErrClaimChanged, err)
	}
	if !sameFileSnapshot(inspection.snapshot.info, before) {
		return fmt.Errorf("%w: declaration file identity differs", ErrClaimChanged)
	}
	raw, opened, err := read(inspection.DeclarationPath, "project declaration", domain.MaxProjectDeclarationBytes)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrClaimChanged, err)
	}
	after, err := os.Lstat(inspection.DeclarationPath)
	if err == nil {
		err = ensureSafeRegularPath(inspection.WorktreeRoot, inspection.DeclarationPath, after)
	}
	if err != nil || !sameFileSnapshot(before, opened) || !sameFileSnapshot(opened, after) {
		return fmt.Errorf("%w: declaration changed while it was read", ErrClaimChanged)
	}
	if !bytes.Equal(raw, inspection.snapshot.raw) || domain.DigestCanonicalJSON(raw) != inspection.snapshot.digest {
		return fmt.Errorf("%w: declaration bytes differ", ErrClaimChanged)
	}
	declaration, err := domain.DecodeProjectDeclaration(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrClaimChanged, err)
	}
	frozen, err := domain.FreezeProjectDeclaration(declaration)
	if err != nil || !bytes.Equal(raw, frozen.CanonicalJSON()) {
		return fmt.Errorf("%w: declaration is no longer canonical", ErrClaimChanged)
	}
	return nil
}

// GuardDependentWrite performs the just-in-time claim check before invoking
// the first dependent write. The callback is never called for unmanaged,
// invalid, or substituted declarations.
func (inspection Inspection) GuardDependentWrite(write func() error) error {
	if write == nil {
		return fmt.Errorf("dependent write callback is required")
	}
	if err := inspection.Revalidate(); err != nil {
		return err
	}
	return write()
}

func invalidInspection(inspection Inspection, reason ClaimReason, err error) Inspection {
	inspection.State = ClaimDeclaredInvalid
	inspection.Reason = reason
	if err != nil {
		inspection.Detail = err.Error()
	}
	return inspection
}

func ensureSafeRegularPath(root, target string, targetInfo os.FileInfo) error {
	if !pathContained(root, target) {
		return fmt.Errorf("path is outside the worktree")
	}
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return fmt.Errorf("resolve declaration path: %w", err)
	}
	current := root
	parts := strings.Split(relative, string(filepath.Separator))
	for index, part := range parts {
		current = filepath.Join(current, part)
		info := targetInfo
		if index != len(parts)-1 {
			info, err = os.Lstat(current)
			if err != nil {
				return fmt.Errorf("lstat declaration ancestor %s: %w", current, err)
			}
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("declaration path contains a symbolic link")
		}
		if index != len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("declaration ancestor is not a directory")
		}
	}
	if !targetInfo.Mode().IsRegular() {
		return fmt.Errorf("project declaration is not a regular file")
	}
	return nil
}

func pathContained(root, target string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func sameFileSnapshot(left, right os.FileInfo) bool {
	if left == nil || right == nil || !os.SameFile(left, right) {
		return false
	}
	return left.Mode() == right.Mode() && left.Size() == right.Size() && left.ModTime().Equal(right.ModTime())
}
