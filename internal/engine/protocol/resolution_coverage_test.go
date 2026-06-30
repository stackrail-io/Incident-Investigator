package protocol

import (
	_ "embed"
	"regexp"
	"testing"

	"github.com/stackrail/incident-investigator/internal/archetype/builtin"
)

//go:embed resolution.go
var resolutionSource string

func TestSeedQuestionIDsHaveResolverCases(t *testing.T) {
	cases := resolverCaseIDs()
	for _, seed := range builtin.DefaultRegistry().SeedQuestions() {
		if !cases[seed.ID] {
			t.Errorf("seed question %q has no explicit case in resolution.go", seed.ID)
		}
	}
}

func resolverCaseIDs() map[string]bool {
	re := regexp.MustCompile(`case "([^"]+)":`)
	matches := re.FindAllStringSubmatch(resolutionSource, -1)
	ids := make(map[string]bool, len(matches))
	for _, m := range matches {
		ids[m[1]] = true
	}
	return ids
}
