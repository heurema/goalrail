package main

import (
	"fmt"
	"time"

	openspecadapter "github.com/heurema/goalrail/internal/adapters/openspec"
	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/harness"
)

const rulesDisclosure = "These rules were present when the schema was replaced; Goalrail neither interprets nor edits them."
const noRulesDisclosure = "No rules were present when the schema was replaced; Goalrail neither added nor interpreted any."
const uncountableRulesDisclosure = "A rules block was present when the schema was replaced; Goalrail could not count its contents and neither interprets nor edits them."

type adoptionReport struct {
	ReplacedSchema   string                           `json:"replaced_schema"`
	AdoptedAt        time.Time                        `json:"adopted_at"`
	SchemaDifference harness.SchemaDifference         `json:"schema_difference"`
	Rules            harness.RulesSnapshot            `json:"rules"`
	RulesDisclosure  string                           `json:"rules_disclosure"`
	Pins             *openspecadapter.SchemaPinCounts `json:"pins,omitempty"`
	SchemaDirectory  string                           `json:"schema_directory"`
	Notices          []string                         `json:"notices,omitempty"`
}

// buildAdoptionReport runs after the overlay and configuration writes. Every
// diagnostic is therefore fail-open: a fact that cannot be produced becomes a
// notice and never turns a completed adoption into a half-installed command.
func buildAdoptionReport(repositoryRoot string, config harness.ConfigOutcome) (*adoptionReport, *ambient.Adoption) {
	if config.Action != harness.ConfigSwitched || config.PreviousSchema == "" {
		return nil, nil
	}
	rules := harness.RulesSnapshot{Counted: true}
	if config.Rules != nil {
		rules = *config.Rules
	}
	report := &adoptionReport{
		ReplacedSchema:   config.PreviousSchema,
		SchemaDifference: harness.CompareRepositorySchemas(repositoryRoot, config.PreviousSchema),
		Rules:            rules,
		RulesDisclosure:  rulesDisclosure,
	}
	if !rules.Counted {
		report.RulesDisclosure = uncountableRulesDisclosure
	} else if !rules.HasRules {
		report.RulesDisclosure = noRulesDisclosure
	}
	if !report.SchemaDifference.Comparable && report.SchemaDifference.Reason != "" {
		report.Notices = append(report.Notices,
			"the schema difference could not be produced: "+report.SchemaDifference.Reason)
	}
	if !rules.Counted && rules.CountReason != "" {
		report.Notices = append(report.Notices, "the rules count could not be produced: "+rules.CountReason)
	}

	pins, err := openspecadapter.CountSchemaPins(repositoryRoot, config.PreviousSchema)
	if err != nil {
		report.Notices = append(report.Notices, "the previous-schema pin count could not be produced: "+err.Error())
		report.SchemaDirectory = "whether the previous schema directory is still pinned could not be determined"
	} else {
		report.Pins = &pins
		if pins.Total() > 0 {
			report.SchemaDirectory = fmt.Sprintf(
				"%d change(s) still name %q; its schema directory must remain",
				pins.Total(), config.PreviousSchema,
			)
		} else {
			report.SchemaDirectory = fmt.Sprintf(
				"no change still names %q; its schema directory may be removed, but Goalrail did not remove it",
				config.PreviousSchema,
			)
		}
	}
	return report, &ambient.Adoption{
		ReplacedSchema: config.PreviousSchema,
		RulesDigest:    rules.Digest,
		HadRules:       rules.HasRules || (rules.Present && !rules.Counted),
	}
}
