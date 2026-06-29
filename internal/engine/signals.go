package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stackrail/incident-investigator/internal/model"
)

// signalKeywords groups raw keywords under a logical signal name. The reasoner
// scans every piece of evidence (summary + stringified payload) for these.
var signalKeywords = map[string][]string{
	"deployment": {"deploy", "deployment", "release", "rollout", "ship", "canary"},
	"rollback":   {"rollback", "rolled back", "reverted", "roll back"},
	"recovery":   {"back to normal", "service recovered", "incident resolved", "fully restored"},
	"database":   {"database", "postgres", "mysql", "sql", "connection pool", "db connection", "query timeout", "deadlock", "replica", "saturat"},
	"config":     {"config", "configuration", "env var", "environment variable", "feature flag", "secret", "parameter"},
	"dns":        {"dns", "name resolution", "nxdomain", "resolve host", "could not resolve"},
	"network":    {"network", "packet loss", "connection refused", "connection reset", "unreachable", "tcp", "socket"},
	"cert":       {"certificate", "tls", "ssl", "x509", "handshake", "expired cert"},
	"memory":     {"memory", "oom", "out of memory", "heap", "leak", "rss"},
	"retry":      {"retry", "retries", "retrying", "thundering herd", "retry storm", "retry budget"},
	"restart":    {"restart", "crashloop", "crashloopbackoff", "oomkilled", "killed", "evicted", "reboot"},
	"latency":    {"latency", "slow", "p99", "p95", "timeout", "timed out", "degraded"},
	"error":      {"error", "errors", "500", "503", "502", "5xx", "exception", "failed", "failure", "panic"},
}

// Signals is a distilled, vendor-neutral summary of what the evidence implies.
// It is the single source of truth that every downstream engine reads from, so
// the heuristics stay consistent with one another.
type Signals struct {
	Categories map[model.Category]int
	Keywords   map[string]bool

	// FirstDeployment is the earliest deployment-related evidence.
	FirstDeployment *model.Evidence
	// IncidentOnset is the earliest evidence that looks like a symptom
	// (an alert, or an error/latency log).
	IncidentOnset *model.Evidence
	// Recovery is the earliest evidence that looks like recovery/rollback.
	Recovery *model.Evidence

	// DeployBeforeIncident is true when a deployment plausibly preceded the
	// first symptom (the classic "bad deploy" temporal signature).
	DeployBeforeIncident bool
	// DeployAfterIncident is true when the only deployment happened after the
	// incident already started, which contradicts a deploy-caused theory.
	DeployAfterIncident bool

	// Entities is the set of distinct entities seen, with occurrence counts.
	Entities map[string]int

	// Lock holds row/table lock-contention signals from database_events.
	Lock LockContention
}

func newSignals() Signals {
	return Signals{
		Categories: map[model.Category]int{},
		Keywords:   map[string]bool{},
		Entities:   map[string]int{},
	}
}

// haystack flattens an evidence record into a single lowercase string for
// keyword scanning.
func haystack(e *model.Evidence) string {
	var b strings.Builder
	b.WriteString(strings.ToLower(e.Summary))
	b.WriteByte(' ')
	b.WriteString(strings.ToLower(string(e.Category)))
	b.WriteByte(' ')
	b.WriteString(strings.ToLower(e.Entity))
	for k, v := range e.Payload {
		b.WriteByte(' ')
		b.WriteString(strings.ToLower(k))
		b.WriteByte(' ')
		b.WriteString(strings.ToLower(fmt.Sprint(v)))
	}
	return b.String()
}

func matchesAny(text string, words []string) bool {
	for _, w := range words {
		if strings.Contains(text, w) {
			return true
		}
	}
	return false
}

// looksLikeSymptom reports whether the evidence reads like an incident symptom.
func looksLikeSymptom(e *model.Evidence, text string) bool {
	switch e.Category {
	case model.CategoryAlertEvents:
		return true
	case model.CategoryApplicationLogs, model.CategoryMetrics, model.CategoryTraceEvents:
		return matchesAny(text, signalKeywords["error"]) || matchesAny(text, signalKeywords["latency"])
	case model.CategoryDeploymentEvents, model.CategoryConfigurationChanges:
		// Deployments and config changes are potential causes, not symptoms.
		return false
	default:
		return matchesAny(text, signalKeywords["error"])
	}
}

// Analyze walks the evidence (assumed roughly chronological after sorting) and
// extracts the high-level signals used across the engine.
func Analyze(s *model.Session) Signals {
	sig := newSignals()

	ordered := sortedByTime(s.Evidence)
	for _, e := range ordered {
		sig.Categories[e.Category]++
		if e.Entity != "" {
			sig.Entities[e.Entity]++
		}
		text := haystack(e)
		for name, words := range signalKeywords {
			if matchesAny(text, words) {
				sig.Keywords[name] = true
			}
		}

		isRecovery := matchesAny(text, signalKeywords["rollback"]) || matchesAny(text, signalKeywords["recovery"])
		isDeploy := !isRecovery && (e.Category == model.CategoryDeploymentEvents || matchesAny(text, signalKeywords["deployment"]))

		if isDeploy && sig.FirstDeployment == nil {
			sig.FirstDeployment = e
		}
		if isRecovery && sig.Recovery == nil {
			sig.Recovery = e
		}
		// Recovery should not be counted as the incident onset.
		if !isRecovery && sig.IncidentOnset == nil && looksLikeSymptom(e, text) {
			sig.IncidentOnset = e
		}
	}

	if sig.FirstDeployment != nil && sig.IncidentOnset != nil {
		if !sig.FirstDeployment.Timestamp.After(sig.IncidentOnset.Timestamp) {
			sig.DeployBeforeIncident = true
		} else {
			sig.DeployAfterIncident = true
		}
	}

	sig.Lock = analyzeLockContention(ordered)

	return sig
}

// sortedByTime returns a copy of the evidence sorted by timestamp (ties broken
// by id for determinism).
func sortedByTime(in []*model.Evidence) []*model.Evidence {
	out := make([]*model.Evidence, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Timestamp.Equal(out[j].Timestamp) {
			return out[i].ID < out[j].ID
		}
		return out[i].Timestamp.Before(out[j].Timestamp)
	})
	return out
}
