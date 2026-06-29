package builtin

import "github.com/stackrail/incident-investigator/internal/archetype"

// DefaultRegistry returns the built-in archetype library.
func DefaultRegistry() *archetype.Registry {
	return archetype.NewRegistry(
		DeploymentCaused{},
		DeploymentUnrelated{},
		DatabaseSaturation{},
		LockContention{},
		ConfigurationChange{},
		NetworkDNS{},
		CertificateExpiry{},
		ResourceExhaustion{},
		RetryStorm{},
		DependencyFailure{},
		PerformanceRegression{},
		ExternalOutage{},
		AuthFailure{},
		HumanError{},
		CapacityPlanning{},
		SecurityIncident{},
		Unknown{},
	)
}

// alwaysApplicable is the Phase 1 default: every built-in archetype competes.
func alwaysApplicable(archetype.ScoreContext) bool { return true }
