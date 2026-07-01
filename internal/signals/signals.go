package signals

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
)

// Keywords groups raw keywords under a logical signal name.
var Keywords = map[string][]string{
	"deployment": {"deploy", "deployment", "release", "rollout", "ship", "canary"},
	"rollback":   {"rollback", "rolled back", "reverted", "roll back"},
	"recovery":   {"back to normal", "service recovered", "incident resolved", "fully restored"},
	"database":   {"database", "postgres", "mysql", "sql", "connection pool", "db connection", "query timeout", "deadlock", "replica", "saturat"},
	"config":     {"config", "configuration", "env var", "environment variable", "secret", "parameter", "configmap"},
	"dns":        {"dns", "name resolution", "nxdomain", "resolve host", "could not resolve"},
	"network":    {"network", "packet loss", "connection refused", "connection reset", "unreachable", "tcp", "socket"},
	"cert":       {"certificate", "tls", "ssl", "x509", "handshake", "expired cert"},
	"memory":     {"memory", "oom", "out of memory", "heap", "leak", "rss"},
	"retry":      {"retry", "retries", "retrying", "thundering herd", "retry storm", "retry budget"},
	"restart":    {"restart", "crashloop", "crashloopbackoff", "oomkilled", "killed", "evicted", "reboot"},
	"latency":    {"latency", "slow", "p99", "p95", "timeout", "timed out", "degraded"},
	"error":      {"error", "errors", "500", "503", "502", "5xx", "exception", "failed", "failure", "panic"},
	"dependency": {"downstream", "upstream", "dependency", "depends on", "caller timeout", "service unavailable"},
	"external":   {"third-party", "third party", "vendor", "saas", "external service", "status page", "provider outage"},
	"auth":       {"oauth", "jwt", "iam", "ldap", "rbac", "unauthorized", "401", "403", "token expired", "authentication failed"},
	"human":      {"manual", "operator", "runbook", "human error", "mistake", "wrong command", "manual change"},
	"capacity":   {"autoscaling", "auto-scaling", "traffic spike", "quota exceeded", "scaling delayed", "throttl"},
	"security":   {"unauthorized access", "breach", "compromise", "exploit", "intrusion", "credential leak"},
	"performance":  {"regression", "latency regression", "slower than baseline", "performance degraded"},
	"infrastructure": {"hypervisor", "cloud event", "node unavailable", "node failure", "hardware alert", "instance reboot", "host failure", "bare metal"},
	"kubernetes":     {"kubernetes", "k8s", "kubelet", "scheduler", "readiness probe", "liveness probe", "pending pod", "replicaset"},
	"container":      {"container", "docker", "oci runtime", "image pull", "pull image", "registry unreachable", "containerd"},
	"storage":        {"storage", "ebs", "pvc", "disk full", "mount failed", "iops", "san", "nas", "volume attach"},
	"cache":          {"redis", "memcached", "cache miss", "cache stampede", "hit ratio", "cache unavailable", "cache eviction"},
	"messaging":      {"kafka", "rabbitmq", "sqs", "pulsar", "consumer lag", "queue buildup", "poison message", "message queue"},
	"loadbalancer":   {"load balancer", "envoy", "nginx", "alb", "haproxy", "proxy failure", "health check failed", "backend unhealthy"},
	"apicontract":    {"schema change", "breaking change", "version mismatch", "serialization error", "deserialization", "invalid payload"},
	"datacorruption": {"corruption", "checksum mismatch", "partial write", "data integrity", "corrupted data"},
	"clock":          {"ntp", "clock drift", "time skew", "skewed clock", "time sync"},
	"featureflag":    {"feature flag", "flag rollout", "flag enabled", "launchdarkly", "gradual rollout", "flag disabled"},
	"regional":       {"availability zone", "az failure", "zone failure", "region outage", "regional failure", "single region"},
	"dr":             {"failover", "disaster recovery", "split brain", "dr drill", "recovery incomplete", "failover failed"},
	"observability":  {"missing metrics", "missing logs", "missing traces", "blind spot", "agent down", "observability gap", "sampling dropped"},
}

// LockContention captures vendor-neutral signals of row/table lock waiting.
type LockContention struct {
	Present                bool
	Entity                 string
	HolderIDs              []string
	WaiterIDs              []string
	SerializedRelease      bool
	MissingLockTimeouts    bool
	HealthyDatabaseMetrics bool
}

// Signals is a distilled, vendor-neutral summary of what the evidence implies.
type Signals struct {
	Categories map[model.Category]int
	Keywords   map[string]bool

	FirstDeployment *model.Evidence
	IncidentOnset   *model.Evidence
	Recovery        *model.Evidence

	DeployBeforeIncident bool
	DeployAfterIncident  bool

	Entities map[string]int
	Lock     LockContention
}

// Analyze walks session evidence and extracts high-level signals.
func Analyze(s *model.Session) Signals {
	sig := Signals{
		Categories: map[model.Category]int{},
		Keywords:   map[string]bool{},
		Entities:   map[string]int{},
	}

	ordered := SortedByTime(s.Evidence)
	for _, e := range ordered {
		sig.Categories[e.Category]++
		if e.Entity != "" {
			sig.Entities[e.Entity]++
		}
		text := Haystack(e)
		rulesOutDeploy := EvidenceRulesOutDeploy(e)
		rulesOutConfig := EvidenceRulesOutConfig(e)
		for name, words := range Keywords {
			if name == "deployment" && rulesOutDeploy {
				continue
			}
			if name == "config" && rulesOutConfig {
				continue
			}
			if MatchesAny(text, words) {
				sig.Keywords[name] = true
			}
		}

		isRecovery := MatchesAny(text, Keywords["rollback"]) || MatchesAny(text, Keywords["recovery"])
		isDeploy := !isRecovery && !rulesOutDeploy && e.Category == model.CategoryDeploymentEvents
		if !isDeploy && e.Category != model.CategoryConfigurationChanges && !rulesOutDeploy {
			isDeploy = MatchesAny(text, Keywords["deployment"])
		}

		if isDeploy && sig.FirstDeployment == nil {
			sig.FirstDeployment = e
		}
		if isRecovery && sig.Recovery == nil {
			sig.Recovery = e
		}
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

// Haystack flattens evidence into a lowercase string for keyword scanning.
func Haystack(e *model.Evidence) string {
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

// MatchesAny reports whether text contains any keyword.
func MatchesAny(text string, words []string) bool {
	for _, w := range words {
		if strings.Contains(text, w) {
			return true
		}
	}
	return false
}

// EvidenceMatching returns de-duplicated evidence ids satisfying pred.
func EvidenceMatching(s *model.Session, pred func(*model.Evidence) bool) []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range SortedByTime(s.Evidence) {
		if pred(e) && !seen[e.ID] {
			seen[e.ID] = true
			out = append(out, e.ID)
		}
	}
	return out
}

// MentionsMissingTimeout reports absent statement/lock timeout configuration.
func MentionsMissingTimeout(text string) bool {
	hasTimeout := strings.Contains(text, "lock_timeout") || strings.Contains(text, "statement_timeout")
	if !hasTimeout {
		return false
	}
	return strings.Contains(text, "no ") ||
		strings.Contains(text, "without") ||
		strings.Contains(text, "missing") ||
		strings.Contains(text, "not set") ||
		strings.Contains(text, "absent") ||
		strings.Contains(text, "unbounded")
}

// LooksLikeHealthyDatabaseMetrics reports non-saturated database metrics.
func LooksLikeHealthyDatabaseMetrics(e *model.Evidence) bool {
	text := Haystack(e)
	isDB := MatchesAny(text, Keywords["database"]) ||
		strings.Contains(text, "postgres") ||
		strings.Contains(text, "mysql") ||
		strings.Contains(text, "sql")
	if !isDB {
		return false
	}
	if strings.Contains(text, "saturat") || strings.Contains(text, "exhaust") {
		return false
	}
	if strings.Contains(text, "100/100") || strings.Contains(text, "cpu 9") || strings.Contains(text, "cpu 8") {
		return false
	}
	return true
}

// SortedByTime returns evidence sorted by timestamp (ties broken by id).
func SortedByTime(in []*model.Evidence) []*model.Evidence {
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

func looksLikeSymptom(e *model.Evidence, text string) bool {
	switch e.Category {
	case model.CategoryAlertEvents:
		return true
	case model.CategoryApplicationLogs, model.CategoryMetrics, model.CategoryTraceEvents:
		return MatchesAny(text, Keywords["error"]) || MatchesAny(text, Keywords["latency"])
	case model.CategoryDeploymentEvents, model.CategoryConfigurationChanges:
		return false
	default:
		return MatchesAny(text, Keywords["error"])
	}
}

const lockReleaseWindow = 2 * time.Second

func analyzeLockContention(ordered []*model.Evidence) LockContention {
	lc := LockContention{}

	byEntity := map[string][]*model.Evidence{}
	for _, e := range ordered {
		if e.Category != model.CategoryDatabaseEvents || e.Entity == "" {
			continue
		}
		byEntity[e.Entity] = append(byEntity[e.Entity], e)
	}

	for entity, evs := range byEntity {
		if len(evs) < 2 {
			continue
		}
		for _, cluster := range clusterByTimestamp(evs, lockReleaseWindow) {
			if len(cluster) < 2 {
				continue
			}
			holders, waiters := partitionLockQueue(cluster)
			if len(holders) == 0 || len(waiters) == 0 {
				continue
			}
			lc.Present = true
			lc.Entity = entity
			lc.SerializedRelease = true
			for _, h := range holders {
				lc.HolderIDs = append(lc.HolderIDs, h.ID)
			}
			for _, w := range waiters {
				lc.WaiterIDs = append(lc.WaiterIDs, w.ID)
			}
			break
		}
		if lc.Present {
			break
		}
	}

	for _, e := range ordered {
		if e.Category != model.CategoryConfigurationChanges {
			continue
		}
		if MentionsMissingTimeout(Haystack(e)) {
			lc.MissingLockTimeouts = true
			break
		}
	}

	for _, e := range ordered {
		if e.Category != model.CategoryMetrics {
			continue
		}
		if LooksLikeHealthyDatabaseMetrics(e) {
			lc.HealthyDatabaseMetrics = true
			break
		}
	}

	return lc
}

func clusterByTimestamp(evs []*model.Evidence, window time.Duration) [][]*model.Evidence {
	if len(evs) == 0 {
		return nil
	}
	sorted := make([]*model.Evidence, len(evs))
	copy(sorted, evs)
	sortSliceByTime(sorted)

	var clusters [][]*model.Evidence
	start := 0
	for i := 1; i <= len(sorted); i++ {
		if i == len(sorted) || sorted[i].Timestamp.Sub(sorted[start].Timestamp) > window {
			if i-start >= 2 {
				clusters = append(clusters, sorted[start:i])
			}
			start = i
		}
	}
	return clusters
}

func sortSliceByTime(evs []*model.Evidence) {
	for i := 1; i < len(evs); i++ {
		for j := i; j > 0 && evs[j].Timestamp.Before(evs[j-1].Timestamp); j-- {
			evs[j], evs[j-1] = evs[j-1], evs[j]
		}
	}
}

func partitionLockQueue(evs []*model.Evidence) (holders, waiters []*model.Evidence) {
	for _, e := range evs {
		rows, ok := payloadInt(e.Payload, "rows_affected")
		if !ok {
			continue
		}
		if rows > 0 {
			holders = append(holders, e)
		} else if rows == 0 {
			waiters = append(waiters, e)
		}
	}
	return holders, waiters
}

func payloadInt(payload map[string]any, key string) (int, bool) {
	if payload == nil {
		return 0, false
	}
	v, ok := payload[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
