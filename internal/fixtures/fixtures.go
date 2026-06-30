// Package fixtures provides realistic, vendor-neutral incident scenarios used to
// exercise the planner, investigation graph, timeline, hypotheses and confidence
// engines. Scenarios are loaded from spec/investigation-v1/conformance/archetype-fixtures/
// declared in archetypes.yaml.
package fixtures

import (
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
)

// Fixture is a complete, replayable incident scenario.
type Fixture struct {
	Name     string
	Question string
	Service  string
	Window   model.TimeWindow
	// Batches are submitted in order, simulating incremental evidence gathering.
	Batches [][]*model.Evidence
	// ExpectLeading is the hypothesis id expected to lead after all batches.
	ExpectLeading string
}

// All returns every archetype conformance scenario from archetypes.yaml.
func All() []Fixture {
	fx, err := AllFromSpec()
	if err != nil {
		panic(err)
	}
	return fx
}

// BadDeployment is the canonical deployment-caused incident scenario.
func BadDeployment() Fixture { return mustLoad("deployment-failure") }

// DatabaseOutage is a database saturation scenario.
func DatabaseOutage() Fixture { return mustLoad("database-saturation") }

// CertificateExpiry is a TLS certificate expiry scenario.
func CertificateExpiry() Fixture { return mustLoad("certificate-tls-failure") }

// DNSOutage is a DNS resolution failure scenario.
func DNSOutage() Fixture { return mustLoad("dns-failure") }

// ConfigurationDrift is a configuration-change scenario (ConfigMap / env var mistake).
func ConfigurationDrift() Fixture { return mustLoad("configuration-drift") }

// KubernetesFailure is a Kubernetes scheduling and readiness scenario.
func KubernetesFailure() Fixture { return mustLoad("kubernetes-failure") }

// Deprecated: use ConfigurationDrift for config mistakes or KubernetesFailure for pod lifecycle faults.
func KubernetesRestartLoop() Fixture { return ConfigurationDrift() }

// MemoryLeak is a resource-exhaustion / OOM scenario.
func MemoryLeak() Fixture { return mustLoad("resource-exhaustion") }

// RetryStorm is a retry amplification scenario.
func RetryStorm() Fixture { return mustLoad("retry-storm") }

// LockContention is a database lock-wait queue scenario.
func LockContention() Fixture { return mustLoad("database-lock-contention") }

// DependencyFailure is a downstream dependency timeout scenario.
func DependencyFailure() Fixture { return mustLoad("dependency-failure") }

// ExternalOutage is a third-party provider outage scenario.
func ExternalOutage() Fixture { return mustLoad("external-outage") }

// base is retained for tests that construct custom evidence relative to fixture time.
var base = time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)

// At returns a timestamp offset from the shared fixture base time.
func At(min, sec int) time.Time {
	return base.Add(time.Duration(min)*time.Minute + time.Duration(sec)*time.Second)
}

// Ev builds a single evidence item for ad-hoc test scenarios.
func Ev(id string, t time.Time, cat model.Category, entity, summary string, payload map[string]any) *model.Evidence {
	return &model.Evidence{
		ID:        id,
		Timestamp: t,
		Category:  cat,
		Source:    "provided_by_client",
		Entity:    entity,
		Summary:   summary,
		Payload:   payload,
	}
}
