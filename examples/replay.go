package examples

import (
	"context"

	"github.com/stackrail/incident-investigator/internal/model"
	"github.com/stackrail/incident-investigator/internal/runtime"
)

// RunReport replays an example investigation and returns the finished report.
func RunReport(ctx context.Context, repoRoot, name string) (model.Report, error) {
	sc, err := Load(repoRoot, name)
	if err != nil {
		return model.Report{}, err
	}
	goal, window, err := sc.StartInput()
	if err != nil {
		return model.Report{}, err
	}
	rt := runtime.New()
	sess, err := rt.Start(ctx, runtime.StartInput{
		Question: sc.Investigation.Question, Service: sc.Investigation.Service,
		TimeWindow: window, Goal: goal,
	})
	if err != nil {
		return model.Report{}, err
	}
	for _, batch := range sc.Batches {
		sess, err = rt.Submit(ctx, sess.ID, batch)
		if err != nil {
			return model.Report{}, err
		}
	}
	report, _, err := rt.Finish(ctx, sess.ID)
	return report, err
}

// AllEvidence returns every evidence item across batches in submission order.
func (s *Scenario) AllEvidence() []*model.Evidence {
	if s == nil {
		return nil
	}
	var out []*model.Evidence
	for _, batch := range s.Batches {
		out = append(out, batch...)
	}
	return out
}
