package model

// Category is a vendor-neutral classification of a piece of evidence.
//
// The engine deliberately knows nothing about CloudWatch, Datadog, Kubernetes,
// or any other system. It only understands these abstract categories. The AI
// assistant is responsible for mapping vendor-specific data into one of them.
type Category string

const (
	CategoryApplicationLogs      Category = "application_logs"
	CategoryInfrastructureEvents Category = "infrastructure_events"
	CategoryDeploymentEvents     Category = "deployment_events"
	CategoryAlertEvents          Category = "alert_events"
	CategoryMetrics              Category = "metrics"
	CategoryTraceEvents          Category = "trace_events"
	CategoryConfigurationChanges Category = "configuration_changes"
	CategoryNetworkEvents        Category = "network_events"
	CategoryDatabaseEvents       Category = "database_events"
	CategorySecurityEvents       Category = "security_events"
	CategoryHumanContext         Category = "human_context"
	CategoryCustom               Category = "custom"
)

// AllCategories returns every supported evidence category.
func AllCategories() []Category {
	return []Category{
		CategoryApplicationLogs,
		CategoryInfrastructureEvents,
		CategoryDeploymentEvents,
		CategoryAlertEvents,
		CategoryMetrics,
		CategoryTraceEvents,
		CategoryConfigurationChanges,
		CategoryNetworkEvents,
		CategoryDatabaseEvents,
		CategorySecurityEvents,
		CategoryHumanContext,
		CategoryCustom,
	}
}

// Valid reports whether c is a recognized category.
func (c Category) Valid() bool {
	for _, known := range AllCategories() {
		if c == known {
			return true
		}
	}
	return false
}

// Priority expresses how badly the planner wants a given category of evidence.
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

// Weight maps a priority to a numeric weight used for progress scoring.
func (p Priority) Weight() float64 {
	switch p {
	case PriorityHigh:
		return 3
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 1
	default:
		return 1
	}
}
