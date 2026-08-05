package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/harness"
)

type LegacyMarkerState string
type LegacyOverlayState string

const (
	LegacyMarkerAbsent  LegacyMarkerState = "absent"
	LegacyMarkerValid   LegacyMarkerState = "valid"
	LegacyMarkerInvalid LegacyMarkerState = "invalid"

	LegacyOverlayAbsent   LegacyOverlayState = "absent"
	LegacyOverlayPartial  LegacyOverlayState = "partial"
	LegacyOverlayComplete LegacyOverlayState = "complete"
	LegacyOverlayInvalid  LegacyOverlayState = "invalid"

	legacyAmbientPath    = ".goalrail/ambient.json"
	legacyAmbientSchema  = "goalrail.ambient-marker/v0"
	maxLegacyMarkerBytes = 8 << 10
)

var legacyV018OverlayPaths = func() []string {
	canon := harness.LegacyV018Canon()
	paths := make([]string, 0, len(canon.Files))
	for _, file := range canon.Files {
		paths = append(paths, file.Path)
	}
	return paths
}()

// LegacyV018Evidence is read-only migration diagnosis. It is never project
// identity and never makes Inspect return managed.
type LegacyV018Evidence struct {
	WorktreeRoot string
	Marker       LegacyMarkerState
	Overlay      LegacyOverlayState
	OverlayCanon string
	Detail       string
}

type legacyMarker struct {
	Schema        string    `json:"schema"`
	InitializedAt time.Time `json:"initialized_at"`
}

func DetectLegacyV018(ctx context.Context, start string) (LegacyV018Evidence, error) {
	root, err := ResolveWorktreeRoot(ctx, start)
	if err != nil {
		return LegacyV018Evidence{}, err
	}
	evidence := LegacyV018Evidence{
		WorktreeRoot: root,
		Marker:       LegacyMarkerAbsent,
		Overlay:      LegacyOverlayAbsent,
	}

	markerPath := filepath.Join(root, filepath.FromSlash(legacyAmbientPath))
	markerInfo, markerErr := os.Lstat(markerPath)
	switch {
	case errors.Is(markerErr, fs.ErrNotExist):
	case markerErr != nil:
		evidence.Marker = LegacyMarkerInvalid
		evidence.Detail = markerErr.Error()
	default:
		if err := ensureSafeRegularPath(root, markerPath, markerInfo); err != nil {
			evidence.Marker = LegacyMarkerInvalid
			evidence.Detail = err.Error()
		} else if raw, err := boundedio.ReadRegularFile(markerPath, "legacy ambient marker", maxLegacyMarkerBytes); err != nil {
			evidence.Marker = LegacyMarkerInvalid
			evidence.Detail = err.Error()
		} else if err := validateLegacyMarker(raw); err != nil {
			evidence.Marker = LegacyMarkerInvalid
			evidence.Detail = err.Error()
		} else {
			evidence.Marker = LegacyMarkerValid
		}
	}

	legacyCanon := harness.LegacyV018Canon()
	present := 0
	invalid := false
	for _, canonFile := range legacyCanon.Files {
		relative := canonFile.Path
		path := filepath.Join(root, filepath.FromSlash(relative))
		info, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil || ensureSafeRegularPath(root, path, info) != nil {
			invalid = true
			continue
		}
		raw, readErr := boundedio.ReadRegularFile(path, "legacy OpenSpec overlay", 512<<10)
		if readErr != nil || harness.Digest(raw) != canonFile.Digest {
			invalid = true
			if evidence.Detail == "" {
				if readErr != nil {
					evidence.Detail = readErr.Error()
				} else {
					evidence.Detail = fmt.Sprintf("%s does not match the retained v0.1.8 canon", relative)
				}
			}
			continue
		}
		present++
	}
	switch {
	case invalid:
		evidence.Overlay = LegacyOverlayInvalid
	case present == 0:
		evidence.Overlay = LegacyOverlayAbsent
	case present == len(legacyCanon.Files):
		evidence.Overlay = LegacyOverlayComplete
		evidence.OverlayCanon = legacyCanon.ID
	default:
		evidence.Overlay = LegacyOverlayPartial
	}
	return evidence, nil
}

func validateLegacyMarker(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var marker legacyMarker
	if err := decoder.Decode(&marker); err != nil {
		return fmt.Errorf("decode legacy ambient marker: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("decode legacy ambient marker: trailing JSON")
	}
	if marker.Schema != legacyAmbientSchema || marker.InitializedAt.IsZero() {
		return fmt.Errorf("legacy ambient marker has unsupported identity")
	}
	return nil
}
