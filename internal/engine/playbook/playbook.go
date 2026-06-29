package playbook

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stackrail/incident-investigator/internal/model"
)

// EffectAction is a playbook outcome on a hypothesis.
type EffectAction string

const (
	EffectIncrease EffectAction = "Increase"
	EffectDecrease EffectAction = "Decrease"
)

// PlaybookEffect applies when a question resolves true/false.
type PlaybookEffect struct {
	WhenTrue     bool
	Action       EffectAction
	HypothesisID string
	Amount       float64
}

// PlaybookQuestion is a declarative question definition in a playbook.
type PlaybookQuestion struct {
	ID            string
	Title         string
	Description   string
	Priority      int
	Requires      []model.Category
	DependsOn     []string
	Effects       []PlaybookEffect
	TriggerSignal string
	Generates     []string
}

// Playbook is a declarative investigation playbook.
type Playbook struct {
	ID        string
	Goal      model.InvestigationGoal
	Questions []PlaybookQuestion
}

// Parse parses a simple investigation playbook DSL into a Playbook.
func Parse(id string, goal model.InvestigationGoal, src string) (*Playbook, error) {
	pb := &Playbook{ID: id, Goal: goal}
	var current *PlaybookQuestion

	lines := strings.Split(src, "\n")
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "QUESTION":
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: QUESTION requires an id", i+1)
			}
			pb.Questions = append(pb.Questions, PlaybookQuestion{ID: parts[1], Priority: 50})
			current = &pb.Questions[len(pb.Questions)-1]
		case "TITLE":
			if current == nil {
				return nil, fmt.Errorf("line %d: TITLE outside QUESTION", i+1)
			}
			current.Title = strings.Join(parts[1:], " ")
		case "DESCRIPTION":
			if current == nil {
				return nil, fmt.Errorf("line %d: DESCRIPTION outside QUESTION", i+1)
			}
			current.Description = strings.Join(parts[1:], " ")
		case "PRIORITY":
			if current == nil || len(parts) < 2 {
				return nil, fmt.Errorf("line %d: invalid PRIORITY", i+1)
			}
			n, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("line %d: PRIORITY: %w", i+1, err)
			}
			current.Priority = n
		case "REQUIRES":
			if current == nil {
				return nil, fmt.Errorf("line %d: REQUIRES outside QUESTION", i+1)
			}
			for _, cat := range parts[1:] {
				current.Requires = append(current.Requires, model.Category(cat))
			}
		case "DEPENDS":
			if current == nil || len(parts) < 2 {
				return nil, fmt.Errorf("line %d: invalid DEPENDS", i+1)
			}
			current.DependsOn = append(current.DependsOn, parts[1])
		case "TRIGGER":
			if current == nil || len(parts) < 2 {
				return nil, fmt.Errorf("line %d: invalid TRIGGER", i+1)
			}
			current.TriggerSignal = parts[1]
		case "GENERATES":
			if current == nil {
				return nil, fmt.Errorf("line %d: GENERATES outside QUESTION", i+1)
			}
			current.Generates = append(current.Generates, parts[1:]...)
		case "IF":
			if current == nil || len(parts) < 5 {
				return nil, fmt.Errorf("line %d: invalid IF rule", i+1)
			}
			whenTrue := parts[1] == "TRUE"
			action := EffectAction(parts[2])
			amount, err := strconv.ParseFloat(parts[len(parts)-1], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: effect amount: %w", i+1, err)
			}
			hypID := strings.Join(parts[3:len(parts)-1], " ")
			current.Effects = append(current.Effects, PlaybookEffect{
				WhenTrue:     whenTrue,
				Action:       action,
				HypothesisID: hypID,
				Amount:       amount,
			})
		default:
			if current != nil && current.Title == "" {
				current.Title = line
			}
		}
	}
	return pb, nil
}

// ForGoal returns the default playbook for an investigation goal.
func ForGoal(goal model.InvestigationGoal) (*Playbook, error) {
	switch goal {
	case model.GoalTimeline:
		return Parse("timeline-default", goal, timelinePlaybook)
	case model.GoalBlastRadius:
		return Parse("blast-radius-default", goal, blastRadiusPlaybook)
	default:
		return Parse("root-cause-default", goal, rootCausePlaybook)
	}
}

const rootCausePlaybook = `
# --- Deployment -----------------------------------------------------------
QUESTION deploy-before-errors
TITLE Did deployment happen before errors?
DESCRIPTION Establish temporal ordering between the most recent deployment and the first symptom.
PRIORITY 85
REQUIRES deployment_events application_logs alert_events
IF TRUE Increase hypothesis-deployment-caused 25
IF FALSE Decrease hypothesis-deployment-caused 40
IF FALSE Increase hypothesis-deployment-unrelated 20

QUESTION deploy-after-incident
TITLE Did deployment occur after incident onset?
DESCRIPTION A deployment that lands after symptoms began contradicts deploy-caused theories.
PRIORITY 80
REQUIRES deployment_events alert_events application_logs
DEPENDS deploy-before-errors
IF TRUE Increase hypothesis-deployment-unrelated 30
IF FALSE Decrease hypothesis-deployment-unrelated 15
IF TRUE Decrease hypothesis-deployment-caused 35

QUESTION rollback-restored-service
TITLE Did rollback restore service?
DESCRIPTION Recovery immediately after rollback strongly implicates the preceding deployment.
PRIORITY 75
REQUIRES deployment_events metrics
DEPENDS deploy-before-errors
IF TRUE Increase hypothesis-deployment-caused 15
IF FALSE Decrease hypothesis-deployment-caused 20

# --- Database -------------------------------------------------------------
QUESTION database-healthy
TITLE Was database capacity healthy?
DESCRIPTION Healthy connections and CPU argue against saturation; lock contention can still cause latency.
PRIORITY 70
REQUIRES database_events metrics application_logs
TRIGGER database
IF TRUE Decrease hypothesis-database-saturation 30
IF FALSE Increase hypothesis-database-saturation 35

QUESTION database-saturated
TITLE Were database connections, CPU, or I/O saturated?
DESCRIPTION Pool exhaustion, high CPU, or replica lag indicate capacity saturation rather than lock waiting.
PRIORITY 72
REQUIRES database_events metrics
DEPENDS database-healthy
IF TRUE Increase hypothesis-database-saturation 30
IF FALSE Decrease hypothesis-database-saturation 20
IF TRUE Decrease hypothesis-lock-contention 20

QUESTION lock-contention-queue
TITLE Were database writes blocked on the same row?
DESCRIPTION Multiple statements on one entity completing together indicate a lock queue, not saturation.
PRIORITY 78
REQUIRES database_events trace_events
TRIGGER database
IF TRUE Increase hypothesis-lock-contention 25
IF FALSE Decrease hypothesis-lock-contention 20
IF TRUE Decrease hypothesis-database-saturation 25

QUESTION lock-timeouts-missing
TITLE Are statement or lock timeouts configured?
DESCRIPTION Missing lock_timeout or statement_timeout lets blocked writers wait unbounded behind a holder.
PRIORITY 65
REQUIRES configuration_changes database_events
DEPENDS lock-contention-queue
IF TRUE Increase hypothesis-lock-contention 15
IF FALSE Decrease hypothesis-lock-contention 10

# --- Cascading failures ---------------------------------------------------
QUESTION latency-before-retries
TITLE Did latency begin before retries amplified?
DESCRIPTION Retries that follow latency suggest amplification; latency that follows retries does not.
PRIORITY 60
REQUIRES metrics trace_events
IF TRUE Increase hypothesis-retry-storm 20
IF FALSE Decrease hypothesis-retry-storm 25

QUESTION traffic-shifted
TITLE Did traffic or load shift during the incident?
DESCRIPTION Sudden traffic shifts can expose latent bottlenecks and amplify retries.
PRIORITY 55
REQUIRES metrics trace_events
DEPENDS pods-restarted
IF TRUE Increase hypothesis-retry-storm 15

# --- Configuration and infrastructure -------------------------------------
QUESTION config-changed
TITLE Was configuration changed?
DESCRIPTION Feature flags, env vars, and connection-pool settings are common incident triggers.
PRIORITY 70
REQUIRES configuration_changes deployment_events
TRIGGER config
GENERATES pods-restarted
IF TRUE Increase hypothesis-configuration-change 30
IF FALSE Decrease hypothesis-configuration-change 15

QUESTION pods-restarted
TITLE Did pods restart or crash loop?
DESCRIPTION Restarts after a config change point to misconfiguration or resource limits.
PRIORITY 65
REQUIRES infrastructure_events application_logs
DEPENDS config-changed
TRIGGER restart
GENERATES traffic-shifted
IF TRUE Increase hypothesis-resource-exhaustion 20

QUESTION memory-pressure
TITLE Was memory pressure or OOM observed?
DESCRIPTION Heap growth, OOM kills, and eviction events indicate resource exhaustion.
PRIORITY 68
REQUIRES metrics infrastructure_events application_logs
TRIGGER memory
IF TRUE Increase hypothesis-resource-exhaustion 30
IF FALSE Decrease hypothesis-resource-exhaustion 20

# --- Network and security -------------------------------------------------
QUESTION dns-failure
TITLE Did DNS resolution fail?
DESCRIPTION NXDOMAIN and name-resolution errors sever connectivity to dependencies.
PRIORITY 75
REQUIRES network_events application_logs
TRIGGER dns
IF TRUE Increase hypothesis-network-dns 30
IF FALSE Decrease hypothesis-network-dns 20

QUESTION network-healthy
TITLE Was network connectivity healthy?
DESCRIPTION Absence of network or DNS symptoms argues against connectivity failures.
PRIORITY 60
REQUIRES network_events application_logs
IF TRUE Decrease hypothesis-network-dns 25
IF FALSE Increase hypothesis-network-dns 30

QUESTION certificate-expired
TITLE Did a TLS certificate expire or become invalid?
DESCRIPTION Expired or mis-issued certificates break TLS handshakes across all clients.
PRIORITY 82
REQUIRES security_events application_logs
TRIGGER cert
IF TRUE Increase hypothesis-certificate-expiry 35
IF FALSE Decrease hypothesis-certificate-expiry 25
`

const timelinePlaybook = `
QUESTION incident-onset
When did the incident begin?
REQUIRES alert_events application_logs
PRIORITY 90

QUESTION deploy-timeline
Did a deployment occur near incident onset?
REQUIRES deployment_events alert_events
DEPENDS incident-onset

QUESTION recovery-timeline
When did service recover?
REQUIRES metrics application_logs deployment_events
DEPENDS incident-onset
`

const blastRadiusPlaybook = `
QUESTION affected-services
Which services were affected?
REQUIRES metrics alert_events trace_events
PRIORITY 90

QUESTION customer-impact
Was customer traffic impacted?
REQUIRES metrics trace_events
DEPENDS affected-services

QUESTION api-degradation
Which APIs degraded?
REQUIRES metrics application_logs
DEPENDS affected-services
`
