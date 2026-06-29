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
QUESTION deploy-before-errors
Did deployment happen before errors?
REQUIRES deployment_events application_logs alert_events
IF TRUE Increase hypothesis-deployment-caused 25
IF FALSE Decrease hypothesis-deployment-caused 40

QUESTION rollback-restored-service
Did rollback restore service?
REQUIRES deployment_events metrics
DEPENDS deploy-before-errors
IF TRUE Increase hypothesis-deployment-caused 15
IF FALSE Decrease hypothesis-deployment-caused 20

QUESTION database-healthy
Was database healthy?
REQUIRES database_events metrics application_logs
IF TRUE Decrease hypothesis-database-saturation 30
IF FALSE Increase hypothesis-database-saturation 35

QUESTION lock-contention-queue
Were database writes blocked on the same row?
REQUIRES database_events trace_events
IF TRUE Increase hypothesis-lock-contention 25
IF FALSE Decrease hypothesis-lock-contention 20
IF TRUE Decrease hypothesis-database-saturation 25

QUESTION latency-before-retries
Did latency begin before retries?
REQUIRES metrics trace_events
IF TRUE Increase hypothesis-retry-storm 20
IF FALSE Decrease hypothesis-retry-storm 25

QUESTION config-changed
Was configuration changed?
REQUIRES configuration_changes deployment_events
TRIGGER config
GENERATES pods-restarted
IF TRUE Increase hypothesis-configuration-change 30
IF FALSE Decrease hypothesis-configuration-change 15

QUESTION pods-restarted
Did pods restart?
REQUIRES infrastructure_events application_logs
DEPENDS config-changed
TRIGGER restart
GENERATES traffic-shifted
IF TRUE Increase hypothesis-resource-exhaustion 20

QUESTION traffic-shifted
Did traffic shift?
REQUIRES metrics trace_events
DEPENDS pods-restarted
IF TRUE Increase hypothesis-retry-storm 15

QUESTION network-healthy
Was network healthy?
REQUIRES network_events application_logs
IF TRUE Decrease hypothesis-network-dns 25
IF FALSE Increase hypothesis-network-dns 30
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
