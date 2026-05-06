package projectscan

import (
	"crypto/sha256"
	"encoding/hex"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func BaselineProfileID(repoBindingID string, canonicalRepoRoot string, headSHA string, schemaVersion int) string {
	sum := sha256.Sum256([]byte(repoBindingID + "\n" + canonicalRepoRoot + "\n" + headSHA + "\n" + intString(schemaVersion)))
	return "rbp_" + hex.EncodeToString(sum[:])[:24]
}

func workspaceOverlayID(repoBindingID string, baselineID string, canonicalRepoRoot string, baseHeadSHA string, statusHash string) string {
	sum := sha256.Sum256([]byte(repoBindingID + "\n" + baselineID + "\n" + canonicalRepoRoot + "\n" + baseHeadSHA + "\n" + statusHash + "\n" + intString(SchemaVersion)))
	return "wso_" + hex.EncodeToString(sum[:])[:24]
}

func cacheKey(repoBindingID string, canonicalRepoRoot string) string {
	sum := sha256.Sum256([]byte(repoBindingID + "\n" + canonicalRepoRoot))
	return hex.EncodeToString(sum[:])[:32]
}

func hashStrings(name string, values []string) HashReceipt {
	copied := append([]string(nil), values...)
	sort.Strings(copied)
	sum := sha256.Sum256([]byte(strings.Join(copied, "\n")))
	return HashReceipt{Name: name, Algorithm: "sha256", Value: hex.EncodeToString(sum[:])}
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	negative := value < 0
	if negative {
		value = -value
	}
	for value > 0 {
		i--
		b[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func normalizeRelativePath(value string) string {
	value = strings.TrimSpace(filepath.ToSlash(value))
	if value == "" || value == "." {
		return ""
	}
	value = strings.TrimPrefix(value, "./")
	cleaned := path.Clean(value)
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return ""
	}
	return cleaned
}

func sortStrings(values []string) {
	sort.Strings(values)
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func appendUnique(set map[string]struct{}, value string) {
	if value == "" {
		return
	}
	set[value] = struct{}{}
}

func uniqueSorted(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	return sortedKeys(set)
}
