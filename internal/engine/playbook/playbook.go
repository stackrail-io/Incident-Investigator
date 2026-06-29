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
		return rootCauseFromRegistry(), nil
	}
}

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
