package readiness

const (
	CheckReadmePresent       = "readme_present"
	CheckLicensePresent      = "license_present"
	CheckCIDetected          = "ci_detected"
	CheckTestsDetected       = "tests_detected"
	CheckAgentsRulesDetected = "agents_or_rules_detected"
	CheckCodeownersDetected  = "codeowners_detected"
	CheckLanguageDetected    = "repo_language_detected"
	CheckProofSurface        = "proof_surface_possible"
)

type checkWeight struct {
	name   string
	weight int
}

var checkWeights = []checkWeight{
	{name: CheckReadmePresent, weight: 10},
	{name: CheckLicensePresent, weight: 10},
	{name: CheckCIDetected, weight: 15},
	{name: CheckTestsDetected, weight: 20},
	{name: CheckAgentsRulesDetected, weight: 10},
	{name: CheckCodeownersDetected, weight: 10},
	{name: CheckLanguageDetected, weight: 10},
	{name: CheckProofSurface, weight: 15},
}

func weightFor(check string) int {
	for _, item := range checkWeights {
		if item.name == check {
			return item.weight
		}
	}
	return 0
}
