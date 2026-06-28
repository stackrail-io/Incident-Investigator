package reasoning

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/stackrail/incident-investigator/internal/graph"
	"github.com/stackrail/incident-investigator/internal/model"
)

// HybridOrchestrator runs multiple reasoners, merges actions, validates, and applies.
type HybridOrchestrator struct {
	registry  *Registry
	strategy  model.OrchestrationStrategy
	merger    *Merger
	validator *Validator
	applier   *Applier
	now       func() time.Time
}

// HybridOption configures the hybrid orchestrator.
type HybridOption func(*HybridOrchestrator)

// WithStrategy sets the orchestration strategy.
func WithStrategy(s model.OrchestrationStrategy) HybridOption {
	return func(o *HybridOrchestrator) { o.strategy = s }
}

// WithClock overrides the time source.
func WithClock(now func() time.Time) HybridOption {
	return func(o *HybridOrchestrator) { o.now = now }
}

// NewHybridOrchestrator returns an orchestrator backed by a reasoner registry.
func NewHybridOrchestrator(reg *Registry, opts ...HybridOption) *HybridOrchestrator {
	o := &HybridOrchestrator{
		registry:  reg,
		strategy:  model.StrategySequential,
		merger:    NewMerger(model.StrategySequential),
		validator: NewValidator(),
		applier:   NewApplier(),
		now:       time.Now,
	}
	for _, opt := range opts {
		opt(o)
	}
	o.merger = NewMerger(o.strategy)
	return o
}

// Execute runs all supporting reasoners and applies merged actions.
func (o *HybridOrchestrator) Execute(ctx context.Context, inv *Investigation) (*model.ReasoningCycle, error) {
	started := o.now()
	cycle := &model.ReasoningCycle{
		CycleID:   newID("cycle"),
		StartedAt: started,
		Strategy:  string(o.strategy),
	}

	g := inv.Graph
	if g == nil && inv.Session != nil && inv.Session.Graph != nil {
		g = graph.FromView(inv.Session.Graph)
	}

	reasoners := o.registry.List()
	sort.SliceStable(reasoners, func(i, j int) bool {
		return reasoners[i].Priority() > reasoners[j].Priority()
	})

	var results []model.ReasoningResult
	if o.strategy == model.StrategyParallel {
		results = o.runParallel(ctx, inv, reasoners)
	} else {
		results = o.runSequential(ctx, inv, reasoners)
	}
	cycle.Reasoners = results

	merged := o.merger.Merge(results)
	applyResult := o.applier.Apply(inv.Session, g, merged, o.validator)
	cycle.AppliedActions = applyResult.Applied
	cycle.RejectedActions = applyResult.Rejected

	if inv.Session != nil && inv.Session.Graph != nil {
		inv.Graph = graph.FromView(inv.Session.Graph)
	}

	cycle.CompletedAt = o.now()
	cycle.Duration = cycle.CompletedAt.Sub(started)
	return cycle, nil
}

func (o *HybridOrchestrator) runSequential(ctx context.Context, inv *Investigation, reasoners []Reasoner) []model.ReasoningResult {
	var results []model.ReasoningResult
	for _, r := range reasoners {
		if !r.Supports(inv.Session) {
			continue
		}
		res, err := o.runOne(ctx, inv, r)
		if err != nil {
			continue
		}
		results = append(results, *res)
	}
	return results
}

func (o *HybridOrchestrator) runParallel(ctx context.Context, inv *Investigation, reasoners []Reasoner) []model.ReasoningResult {
	var (
		mu      sync.Mutex
		results []model.ReasoningResult
		wg      sync.WaitGroup
	)
	for _, r := range reasoners {
		if !r.Supports(inv.Session) {
			continue
		}
		wg.Add(1)
		go func(reasoner Reasoner) {
			defer wg.Done()
			res, err := o.runOne(ctx, inv, reasoner)
			if err != nil {
				return
			}
			mu.Lock()
			results = append(results, *res)
			mu.Unlock()
		}(r)
	}
	wg.Wait()
	sort.SliceStable(results, func(i, j int) bool {
		return reasonerPriority(results[i].Reasoner) > reasonerPriority(results[j].Reasoner)
	})
	return results
}

func (o *HybridOrchestrator) runOne(ctx context.Context, inv *Investigation, r Reasoner) (*model.ReasoningResult, error) {
	start := time.Now()
	res, err := r.Analyze(ctx, inv)
	if err != nil {
		return nil, err
	}
	if res == nil {
		res = &model.ReasoningResult{Reasoner: r.Name()}
	}
	res.Reasoner = r.Name()
	res.Metrics.ExecutionTime = time.Since(start)
	res.Metrics.ActionsProposed = len(res.Actions)
	countActionMetrics(res)
	return res, nil
}

func countActionMetrics(res *model.ReasoningResult) {
	for _, a := range res.Actions {
		switch a.Type {
		case model.ActionCreateQuestion:
			res.Metrics.QuestionsCreated++
		case model.ActionResolveQuestion:
			res.Metrics.QuestionsResolved++
		case model.ActionCreateHypothesis:
			res.Metrics.HypothesesCreated++
		case model.ActionRejectHypothesis:
			res.Metrics.HypothesesRejected++
		case model.ActionMarkContradiction:
			res.Metrics.ContradictionsFound++
		case model.ActionCreateRecommendation:
			res.Metrics.RecommendationsCreated++
		case model.ActionLinkGraphNodes:
			res.Metrics.GraphEdgesCreated++
		case model.ActionIncreaseHypothesisConfidence, model.ActionDecreaseHypothesisConfidence:
			res.Metrics.AverageConfidenceDelta += a.Delta
		}
	}
	if n := len(res.Actions); n > 0 && res.Metrics.AverageConfidenceDelta > 0 {
		res.Metrics.AverageConfidenceDelta /= float64(n)
	}
}

func newID(prefix string) string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b[:]))
}
