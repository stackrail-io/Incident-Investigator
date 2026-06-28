package engine

import "github.com/stackrail/incident-investigator/internal/model"

// StateMachine applies deterministic investigation state transitions.
type StateMachine struct {
	HighConfidenceThreshold float64
}

// NewStateMachine returns the default state machine.
func NewStateMachine() *StateMachine {
	return &StateMachine{HighConfidenceThreshold: 70}
}

// Transition computes the next state from current inputs. Completed and failed
// states are terminal.
func (sm *StateMachine) Transition(
	current model.InvestigationState,
	s *model.Session,
	sufficiency model.SufficiencyReport,
) model.InvestigationState {
	if current == model.StateCompleted {
		return model.StateCompleted
	}
	if current == model.StateFailed {
		return model.StateFailed
	}

	if len(s.Evidence) == 0 {
		if current == model.StateStarted {
			return model.StateStarted
		}
		return model.StateCollectingEvidence
	}

	// Reasoning is the transient state during recompute; callers set it before
	// computing derived fields, then this function resolves the outward state.
	if sufficiency.CanAnswer && s.Confidence >= sm.HighConfidenceThreshold {
		return model.StateHighConfidence
	}
	if len(sufficiency.BlockingQuestions) > 0 || len(s.MissingEvidence) > 0 {
		return model.StateWaitingForEvidence
	}
	if s.Confidence >= sm.HighConfidenceThreshold {
		return model.StateHighConfidence
	}
	return model.StateCollectingEvidence
}

// BeginReasoning returns the transient state entered during a recompute cycle.
func (sm *StateMachine) BeginReasoning(current model.InvestigationState) model.InvestigationState {
	if current == model.StateCompleted || current == model.StateFailed {
		return current
	}
	return model.StateReasoning
}
