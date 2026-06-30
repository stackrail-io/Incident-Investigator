package builtin

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
	sigpkg "github.com/stackrail/incident-investigator/internal/signals"
)

// TestKeywordCollisionLeadingHypothesis compares raw Candidate.Score values before
// registry normalization. End-to-end confidence ordering is covered by spec conformance tests.
func TestKeywordCollisionLeadingHypothesis(t *testing.T) {
	now := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		ev     []*model.Evidence
		leader string
	}{
		{
			name: "dns beats network",
			ev: []*model.Evidence{{
				ID: "e1", Timestamp: now, Category: model.CategoryApplicationLogs, Entity: "api",
				Summary: "dns nxdomain; could not resolve host name resolution failure",
			}},
			leader: "hypothesis-dns-failure",
		},
		{
			name: "load balancer beats network",
			ev: []*model.Evidence{{
				ID: "e1", Timestamp: now, Category: model.CategoryApplicationLogs, Entity: "gateway",
				Summary: "nginx health check failed; alb backend unhealthy; envoy dead pool",
			}},
			leader: "hypothesis-load-balancer-failure",
		},
		{
			name: "external beats dependency",
			ev: []*model.Evidence{{
				ID: "e1", Timestamp: now, Category: model.CategoryApplicationLogs, Entity: "payments-api",
				Summary: "third-party payment provider outage; vendor status page reports external service unavailable",
			}},
			leader: "hypothesis-external-outage",
		},
		{
			name: "feature flag beats configuration drift",
			ev: []*model.Evidence{{
				ID: "e1", Timestamp: now, Category: model.CategoryConfigurationChanges, Entity: "api",
				Summary: "feature flag rollout enabled for all users via launchdarkly gradual rollout",
			}},
			leader: "hypothesis-feature-flag-failure",
		},
		{
			name: "retry storm beats performance regression",
			ev: []*model.Evidence{
				{
					ID: "e1", Timestamp: now, Category: model.CategoryApplicationLogs, Entity: "api",
					Summary: "retry storm thundering herd; aggressive retries exceeded retry budget",
				},
				{
					ID: "e2", Timestamp: now.Add(time.Minute), Category: model.CategoryMetrics, Entity: "api",
					Summary: "p95 latency 8s vs 200ms baseline",
				},
			},
			leader: "hypothesis-retry-storm",
		},
		{
			name: "database saturation beats dr failover",
			ev: []*model.Evidence{
				{
					ID: "e1", Timestamp: now, Category: model.CategoryMetrics, Entity: "postgres",
					Summary: "db connections 100/100 saturated, cpu 95%",
				},
				{
					ID: "e2", Timestamp: now.Add(time.Minute), Category: model.CategoryDatabaseEvents, Entity: "postgres",
					Summary: "Postgres active connections at limit; write IOPS saturated",
				},
			},
			leader: "hypothesis-database-saturation",
		},
		{
			name: "data corruption beats human error",
			ev: []*model.Evidence{{
				ID: "e1", Timestamp: now, Category: model.CategoryApplicationLogs, Entity: "ledger",
				Summary: "checksum mismatch detected; corrupted data failed data integrity validation",
			}},
			leader: "hypothesis-data-corruption",
		},
		{
			name: "configuration drift beats kubernetes failure",
			ev: []*model.Evidence{
				{
					ID: "e1", Timestamp: now, Category: model.CategoryConfigurationChanges, Entity: "worker",
					Summary: "Applied new ConfigMap with invalid DB_HOST env var",
				},
				{
					ID: "e2", Timestamp: now.Add(time.Minute), Category: model.CategoryApplicationLogs, Entity: "worker",
					Summary: "fatal: cannot connect using configured DB_HOST",
				},
			},
			leader: "hypothesis-configuration-change",
		},
		{
			name: "kubernetes beats configuration drift",
			ev: []*model.Evidence{{
				ID: "e1", Timestamp: now, Category: model.CategoryInfrastructureEvents, Entity: "worker",
				Summary: "kubernetes pending pod; readiness probe failing on k8s workload; crashloopbackoff",
			}},
			leader: "hypothesis-kubernetes-failure",
		},
		{
			name: "cache beats database saturation",
			ev: []*model.Evidence{{
				ID: "e1", Timestamp: now, Category: model.CategoryApplicationLogs, Entity: "api",
				Summary: "redis cache unavailable; cache stampede; memcached hit ratio collapsed",
			}},
			leader: "hypothesis-cache-failure",
		},
		{
			name: "storage beats resource exhaustion",
			ev: []*model.Evidence{{
				ID: "e1", Timestamp: now, Category: model.CategoryInfrastructureEvents, Entity: "node-1",
				Summary: "ebs volume attach failed; disk full; pvc mount failed; iops exhausted",
			}},
			leader: "hypothesis-storage-failure",
		},
	}

	reg := DefaultRegistry()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &model.Session{
				ID:       tc.name,
				Question: "What happened?",
				Evidence: tc.ev,
				Graph:    model.NewEmptyGraphView(),
			}
			sig := sigpkg.Analyze(s)
			ctx := archetype.ScoreContext{Session: s, Signals: sig}
			hyps := scoreCandidates(reg, ctx)
			got := leadingCandidate(hyps)
			if got != tc.leader {
				t.Errorf("leader = %q, want %q (top scores: %s)", got, tc.leader, formatTopScores(hyps, 3))
			}
		})
	}
}

func scoreCandidates(reg *archetype.Registry, ctx archetype.ScoreContext) []archetype.Candidate {
	var out []archetype.Candidate
	for _, a := range reg.All() {
		if a.Applicable(ctx) {
			out = append(out, a.Score(ctx))
		}
	}
	return out
}

func leadingCandidate(hyps []archetype.Candidate) string {
	bestID := ""
	var best float64
	for _, h := range hyps {
		if h.Score > best {
			best = h.Score
			bestID = h.HypothesisID
		}
	}
	return bestID
}

func formatTopScores(hyps []archetype.Candidate, n int) string {
	type pair struct {
		id    string
		score float64
	}
	var ranked []pair
	for _, h := range hyps {
		if h.Score > 0 {
			ranked = append(ranked, pair{h.HypothesisID, h.Score})
		}
	}
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].score > ranked[i].score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
	if len(ranked) > n {
		ranked = ranked[:n]
	}
	out := ""
	for i, p := range ranked {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%s=%.1f", p.id, p.score)
	}
	return out
}
