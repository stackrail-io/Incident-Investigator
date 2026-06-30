package spec

// EffectiveLeadMargin returns the minimum confidence gap required over the runner-up.
func (fx *ConformanceFixture) EffectiveLeadMargin() float64 {
	if fx == nil {
		return 0
	}
	if fx.ExpectAfterAllEvidence.MinLeadMargin > 0 {
		return fx.ExpectAfterAllEvidence.MinLeadMargin
	}
	if fx.ArchetypeID != "" && fx.ArchetypeID != "unknown-novel" {
		return 3.0
	}
	return 0
}

// EffectiveMustNotLead returns hypotheses that must not rank first.
func (fx *ConformanceFixture) EffectiveMustNotLead() []string {
	if fx == nil {
		return nil
	}
	if len(fx.ExpectAfterAllEvidence.MustNotLead) > 0 {
		return fx.ExpectAfterAllEvidence.MustNotLead
	}
	if fx.ArchetypeID != "" && fx.ArchetypeID != "unknown-novel" {
		return []string{"hypothesis-unknown"}
	}
	return nil
}
