package signals

import (
	"strings"

	"github.com/stackrail/incident-investigator/internal/model"
)

// EvidenceRulesOutDeploy reports whether evidence in deployment_events explicitly
// rules out a deployment as the cause (negative finding), rather than recording one.
func EvidenceRulesOutDeploy(e *model.Evidence) bool {
	if e == nil || e.Category != model.CategoryDeploymentEvents {
		return false
	}
	if payloadNegates(e, "new_deploy", "deployed", "deployment_occurred", "deploy_occurred") {
		return true
	}
	return summaryNegates(Haystack(e),
		"no deployment", "no deploy", "not deployed", "without deployment",
		"deployment ruled out", "ruled out deployment", "no recent deploy",
		"no deployment in", "deploy: false", "new_deploy: false",
	)
}

// EvidenceRulesOutConfig reports whether configuration_changes evidence rules out
// a config change as the cause.
func EvidenceRulesOutConfig(e *model.Evidence) bool {
	if e == nil || e.Category != model.CategoryConfigurationChanges {
		return false
	}
	if payloadNegates(e, "config_changed", "configuration_changed", "config_change") {
		return true
	}
	return summaryNegates(Haystack(e),
		"no config change", "no configuration change", "not config",
		"config ruled out", "ruled out config", "without config change",
		"config_changed: false", "configuration_changed: false",
	)
}

// HasAffirmativeDeploymentEvidence returns true when at least one deployment_events
// item is a positive deployment signal (not a ruled-out finding).
func HasAffirmativeDeploymentEvidence(s *model.Session) bool {
	if s == nil {
		return false
	}
	for _, e := range s.Evidence {
		if e != nil && e.Category == model.CategoryDeploymentEvents && !EvidenceRulesOutDeploy(e) {
			return true
		}
	}
	return false
}

// AllDeploymentEvidenceRuledOut is true when deployment_events were submitted but
// every item explicitly rules out a deployment.
func AllDeploymentEvidenceRuledOut(s *model.Session) bool {
	if s == nil {
		return false
	}
	var n int
	for _, e := range s.Evidence {
		if e == nil || e.Category != model.CategoryDeploymentEvents {
			continue
		}
		n++
		if !EvidenceRulesOutDeploy(e) {
			return false
		}
	}
	return n > 0
}

// HasAffirmativeConfigurationEvidence returns true when configuration evidence
// affirms a change occurred (not a ruled-out finding).
func HasAffirmativeConfigurationEvidence(s *model.Session) bool {
	if s == nil {
		return false
	}
	for _, e := range s.Evidence {
		if e == nil {
			continue
		}
		if e.Category == model.CategoryConfigurationChanges && !EvidenceRulesOutConfig(e) {
			return true
		}
		if e.Category != model.CategoryConfigurationChanges &&
			!EvidenceRulesOutConfig(e) &&
			MatchesAny(Haystack(e), Keywords["config"]) {
			return true
		}
	}
	return false
}

// AllConfigurationEvidenceRuledOut is true when configuration_changes were
// submitted and every item rules out a config change.
func AllConfigurationEvidenceRuledOut(s *model.Session) bool {
	if s == nil {
		return false
	}
	var n int
	for _, e := range s.Evidence {
		if e == nil || e.Category != model.CategoryConfigurationChanges {
			continue
		}
		n++
		if !EvidenceRulesOutConfig(e) {
			return false
		}
	}
	return n > 0
}

func payloadNegates(e *model.Evidence, keys ...string) bool {
	if e == nil || e.Payload == nil {
		return false
	}
	for _, k := range keys {
		v, ok := e.Payload[k]
		if !ok {
			continue
		}
		switch val := v.(type) {
		case bool:
			if !val {
				return true
			}
		case string:
			lower := strings.ToLower(strings.TrimSpace(val))
			if lower == "false" || lower == "no" || lower == "0" {
				return true
			}
		}
	}
	return false
}

func summaryNegates(text string, phrases ...string) bool {
	lower := strings.ToLower(text)
	for _, p := range phrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
