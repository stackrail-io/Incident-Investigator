package spec

import "testing"

func TestEffectiveLeadMargin(t *testing.T) {
	tests := []struct {
		name string
		fx   *ConformanceFixture
		want float64
	}{
		{"nil", nil, 0},
		{"explicit", func() *ConformanceFixture {
			fx := &ConformanceFixture{}
			fx.ExpectAfterAllEvidence.MinLeadMargin = 5
			return fx
		}(), 5},
		{"default core", &ConformanceFixture{ArchetypeID: "deployment-failure"}, 3},
		{"unknown novel", &ConformanceFixture{ArchetypeID: "unknown-novel"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fx.EffectiveLeadMargin(); got != tt.want {
				t.Errorf("EffectiveLeadMargin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEffectiveMustNotLead(t *testing.T) {
	tests := []struct {
		name string
		fx   *ConformanceFixture
		want []string
	}{
		{"nil", nil, nil},
		{"explicit", func() *ConformanceFixture {
			fx := &ConformanceFixture{}
			fx.ExpectAfterAllEvidence.MustNotLead = []string{"hypothesis-a", "hypothesis-b"}
			return fx
		}(), []string{"hypothesis-a", "hypothesis-b"}},
		{"default core", &ConformanceFixture{ArchetypeID: "deployment-failure"}, []string{"hypothesis-unknown"}},
		{"unknown novel", &ConformanceFixture{ArchetypeID: "unknown-novel"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fx.EffectiveMustNotLead()
			if len(got) != len(tt.want) {
				t.Fatalf("EffectiveMustNotLead() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("EffectiveMustNotLead()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
