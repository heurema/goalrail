package readiness

import (
	"bytes"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/heurema/goalrail/apps/cli/internal/spine"
)

type repoIndex struct {
	files map[string]bool
	dirs  map[string]bool
}

func Scan(fsys fs.FS) (spine.ReadinessReport, error) {
	index, err := buildIndex(fsys)
	if err != nil {
		return spine.ReadinessReport{}, err
	}

	findings := []spine.ReadinessFinding{}
	evidence := []spine.ReadinessEvidence{}

	add := func(finding spine.ReadinessFinding, item spine.ReadinessEvidence) {
		findings = append(findings, finding)
		evidence = append(evidence, item)
	}

	readmePaths := index.anyFiles("README.md", "README", "readme.md")
	if len(readmePaths) > 0 {
		add(pass(CheckReadmePresent, "README evidence found"), pathsEvidence(CheckReadmePresent, readmePaths))
	} else {
		add(fail(CheckReadmePresent, "README evidence was not found"), detailEvidence(CheckReadmePresent, "expected README.md, README, or readme.md"))
	}

	licensePaths := index.anyFiles("LICENSE", "LICENSE.md")
	if len(licensePaths) > 0 {
		add(pass(CheckLicensePresent, "license evidence found"), pathsEvidence(CheckLicensePresent, licensePaths))
	} else {
		add(fail(CheckLicensePresent, "license evidence was not found"), detailEvidence(CheckLicensePresent, "expected LICENSE or LICENSE.md"))
	}

	ciPaths := index.matchFiles(func(name string) bool {
		return strings.HasPrefix(name, ".github/workflows/") && name != ".github/workflows/"
	})
	if len(ciPaths) > 0 {
		add(pass(CheckCIDetected, "CI workflow evidence found"), pathsEvidence(CheckCIDetected, ciPaths))
	} else {
		add(fail(CheckCIDetected, "CI workflow evidence was not found"), detailEvidence(CheckCIDetected, "expected files under .github/workflows/"))
	}

	testPaths, hasTests := detectTests(fsys, index)
	if hasTests {
		add(pass(CheckTestsDetected, "test evidence found"), pathsEvidence(CheckTestsDetected, testPaths))
	} else {
		add(fail(CheckTestsDetected, "test evidence was not found"), detailEvidence(CheckTestsDetected, "expected *_test.go, package.json test script, pyproject.toml, pytest.ini, or tests/"))
	}

	agentPaths := index.anyFiles("AGENTS.md", "CLAUDE.md", ".github/copilot-instructions.md")
	if index.hasDir(".cursor/rules") {
		agentPaths = append(agentPaths, ".cursor/rules")
	}
	sort.Strings(agentPaths)
	if len(agentPaths) > 0 {
		add(pass(CheckAgentsRulesDetected, "agent or repository rule evidence found"), pathsEvidence(CheckAgentsRulesDetected, agentPaths))
	} else {
		add(fail(CheckAgentsRulesDetected, "agent or repository rule evidence was not found"), detailEvidence(CheckAgentsRulesDetected, "expected AGENTS.md, CLAUDE.md, .cursor/rules, or .github/copilot-instructions.md"))
	}

	codeownersPaths := index.anyFiles("CODEOWNERS", ".github/CODEOWNERS")
	if len(codeownersPaths) > 0 {
		add(pass(CheckCodeownersDetected, "CODEOWNERS evidence found"), pathsEvidence(CheckCodeownersDetected, codeownersPaths))
	} else {
		add(fail(CheckCodeownersDetected, "CODEOWNERS evidence was not found"), detailEvidence(CheckCodeownersDetected, "expected CODEOWNERS or .github/CODEOWNERS"))
	}

	language, languagePaths := detectLanguage(index)
	if language != "" {
		add(pass(CheckLanguageDetected, "repository language evidence found"), spine.ReadinessEvidence{Check: CheckLanguageDetected, Paths: languagePaths, Detail: language})
	} else {
		add(fail(CheckLanguageDetected, "repository language evidence was not found"), detailEvidence(CheckLanguageDetected, "expected go.mod, package.json, pyproject.toml, or Cargo.toml"))
	}

	if hasTests && len(ciPaths) > 0 {
		add(pass(CheckProofSurface, "tests and CI can support proof evidence"), detailEvidence(CheckProofSurface, "tests and CI evidence are present"))
	} else if hasTests || len(ciPaths) > 0 {
		add(warn(CheckProofSurface, "proof surface is partial because tests or CI evidence is missing"), detailEvidence(CheckProofSurface, "add both tests and CI for stronger proof evidence"))
	} else {
		add(warn(CheckProofSurface, "proof surface is weak because tests and CI evidence are missing"), detailEvidence(CheckProofSurface, "add repeatable tests and CI before relying on proof evidence"))
	}

	score := Score(findings)
	return spine.ReadinessReport{
		Score:                  score,
		Status:                 Status(score),
		Findings:               findings,
		Evidence:               evidence,
		RecommendedNextActions: recommendedNextActions(findings),
	}, nil
}

func buildIndex(fsys fs.FS) (repoIndex, error) {
	index := repoIndex{files: map[string]bool{}, dirs: map[string]bool{}}
	err := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		clean := path.Clean(name)
		if clean == "." {
			return nil
		}
		clean = strings.TrimPrefix(clean, "./")

		if entry.IsDir() {
			if shouldSkipDir(clean) {
				return fs.SkipDir
			}
			index.dirs[clean] = true
			return nil
		}

		index.files[clean] = true
		return nil
	})
	return index, err
}

func shouldSkipDir(name string) bool {
	switch path.Base(name) {
	case ".git", "node_modules", "dist", "build", ".build", "coverage":
		return true
	default:
		return false
	}
}

func detectTests(fsys fs.FS, index repoIndex) ([]string, bool) {
	paths := index.matchFiles(func(name string) bool {
		return strings.HasSuffix(name, "_test.go") || name == "pyproject.toml" || name == "pytest.ini"
	})

	if index.hasDir("tests") {
		paths = append(paths, "tests")
	}

	if index.hasFile("package.json") {
		data, err := fs.ReadFile(fsys, "package.json")
		if err == nil && bytes.Contains(data, []byte("\"test\"")) {
			paths = append(paths, "package.json")
		}
	}

	sort.Strings(paths)
	return paths, len(paths) > 0
}

func detectLanguage(index repoIndex) (string, []string) {
	switch {
	case index.hasFile("go.mod"):
		return "go", []string{"go.mod"}
	case index.hasFile("package.json"):
		return "javascript/typescript", []string{"package.json"}
	case index.hasFile("pyproject.toml"):
		return "python", []string{"pyproject.toml"}
	case index.hasFile("Cargo.toml"):
		return "rust", []string{"Cargo.toml"}
	default:
		return "", nil
	}
}

func recommendedNextActions(findings []spine.ReadinessFinding) []string {
	actions := []string{}
	for _, finding := range findings {
		if finding.Status == spine.FindingStatusPass {
			continue
		}

		switch finding.Check {
		case CheckReadmePresent:
			actions = append(actions, "Add a README that explains the repository purpose and local checks.")
		case CheckLicensePresent:
			actions = append(actions, "Add a license file before sharing repository evidence.")
		case CheckCIDetected:
			actions = append(actions, "Add a CI workflow so proof packets can reference repeatable automation.")
		case CheckTestsDetected:
			actions = append(actions, "Add at least one repeatable test signal.")
		case CheckAgentsRulesDetected:
			actions = append(actions, "Add AGENTS.md or repository rules to make execution expectations explicit.")
		case CheckCodeownersDetected:
			actions = append(actions, "Add CODEOWNERS to make review ownership inspectable.")
		case CheckLanguageDetected:
			actions = append(actions, "Add a standard language manifest such as go.mod, package.json, pyproject.toml, or Cargo.toml.")
		case CheckProofSurface:
			actions = append(actions, "Add both tests and CI before treating proof evidence as strong.")
		}
	}

	if len(actions) == 0 {
		return []string{"Next: validate a working contract with goalrail contract validate --file <contract.json>."}
	}
	return actions
}

func (i repoIndex) hasFile(name string) bool {
	return i.files[name]
}

func (i repoIndex) hasDir(name string) bool {
	return i.dirs[name]
}

func (i repoIndex) anyFiles(names ...string) []string {
	paths := []string{}
	for _, name := range names {
		if i.hasFile(name) {
			paths = append(paths, name)
		}
	}
	sort.Strings(paths)
	return paths
}

func (i repoIndex) matchFiles(match func(string) bool) []string {
	paths := []string{}
	for name := range i.files {
		if match(name) {
			paths = append(paths, name)
		}
	}
	sort.Strings(paths)
	return paths
}

func pass(check, message string) spine.ReadinessFinding {
	return spine.ReadinessFinding{Check: check, Status: spine.FindingStatusPass, Message: message}
}

func warn(check, message string) spine.ReadinessFinding {
	return spine.ReadinessFinding{Check: check, Status: spine.FindingStatusWarn, Message: message}
}

func fail(check, message string) spine.ReadinessFinding {
	return spine.ReadinessFinding{Check: check, Status: spine.FindingStatusFail, Message: message}
}

func pathsEvidence(check string, paths []string) spine.ReadinessEvidence {
	return spine.ReadinessEvidence{Check: check, Paths: paths}
}

func detailEvidence(check, detail string) spine.ReadinessEvidence {
	return spine.ReadinessEvidence{Check: check, Detail: detail}
}
