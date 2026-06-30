//go:build ignore

// One-off generator: go run internal/spec/cmd/gen-archetype-fixtures/main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type fixture struct {
	ArchetypeID  string       `yaml:"archetype_id"`
	ScenarioID   string       `yaml:"scenario_id"`
	Tier         string       `yaml:"tier"`
	Description  string       `yaml:"description"`
	Start        startBlock   `yaml:"start"`
	ExpectStart  expectStart  `yaml:"expect_after_start"`
	Batches      [][]evidence `yaml:"evidence_batches"`
	ExpectAll    expectAll    `yaml:"expect_after_all_evidence"`
	ExpectFinish expectFinish `yaml:"expect_after_finish"`
}

type startBlock struct {
	Question   string     `yaml:"question"`
	Service    string     `yaml:"service"`
	Goal       string     `yaml:"goal"`
	TimeWindow timeWindow `yaml:"time_window"`
}

type timeWindow struct {
	Start string `yaml:"start"`
	End   string `yaml:"end"`
}

type expectStart struct {
	RequiredEvidenceMin int `yaml:"required_evidence_min"`
	PlanQuestionsMin    int `yaml:"plan_questions_min"`
}

type evidence struct {
	ID        string         `yaml:"id"`
	Timestamp string         `yaml:"timestamp"`
	Category  string         `yaml:"category"`
	Entity    string         `yaml:"entity"`
	Summary   string         `yaml:"summary"`
	Payload   map[string]any `yaml:"payload,omitempty"`
}

type expectAll struct {
	LeadingHypothesisID string   `yaml:"leading_hypothesis_id"`
	MinConfidence       float64  `yaml:"min_confidence"`
	MinLeadMargin       float64  `yaml:"min_lead_margin,omitempty"`
	MustNotLead         []string `yaml:"must_not_lead,omitempty"`
	MinHypotheses       int      `yaml:"min_hypotheses"`
	GraphNodesMin       int      `yaml:"graph_nodes_min"`
}

type expectFinish struct {
	State                string   `yaml:"state"`
	ReportRequiredFields []string `yaml:"report_required_fields"`
}

type expectOverride struct {
	margin    float64
	mustNot   []string
}

var conformanceOverrides = map[string]expectOverride{
	"deployment-failure":       {margin: 5, mustNot: []string{"hypothesis-unknown", "hypothesis-deployment-unrelated"}},
	"deployment-unrelated":     {margin: 5, mustNot: []string{"hypothesis-deployment-caused"}},
	"network-failure":          {margin: 3, mustNot: []string{"hypothesis-dns-failure", "hypothesis-unknown"}},
	"dns-failure":              {margin: 5, mustNot: []string{"hypothesis-network-failure", "hypothesis-unknown"}},
	"dependency-failure":     {margin: 3, mustNot: []string{"hypothesis-retry-storm", "hypothesis-unknown"}},
	"performance-regression":   {margin: 3, mustNot: []string{"hypothesis-retry-storm", "hypothesis-unknown"}},
	"feature-flag-failure":     {margin: 3, mustNot: []string{"hypothesis-configuration-change", "hypothesis-unknown"}},
	"configuration-drift":      {margin: 3, mustNot: []string{"hypothesis-kubernetes-failure", "hypothesis-deployment-caused", "hypothesis-unknown"}},
	"kubernetes-failure":       {margin: 3, mustNot: []string{"hypothesis-deployment-caused", "hypothesis-configuration-change", "hypothesis-unknown"}},
	"database-saturation":      {margin: 3, mustNot: []string{"hypothesis-dr-failover-failure", "hypothesis-unknown"}},
	"load-balancer-failure":    {margin: 3, mustNot: []string{"hypothesis-network-failure", "hypothesis-unknown"}},
	"data-corruption":          {margin: 3, mustNot: []string{"hypothesis-human-error", "hypothesis-unknown"}},
	"observability-failure":    {margin: 1, mustNot: []string{"hypothesis-human-error", "hypothesis-unknown"}},
	"external-outage":          {margin: 2, mustNot: []string{"hypothesis-dependency-failure", "hypothesis-unknown"}},
	"unknown-novel":            {margin: 0, mustNot: nil},
}

func enrichExpect(archetypeID string, e expectAll) expectAll {
	if o, ok := conformanceOverrides[archetypeID]; ok {
		e.MinLeadMargin = o.margin
		e.MustNotLead = o.mustNot
	} else {
		e.MinLeadMargin = 3
		e.MustNotLead = []string{"hypothesis-unknown"}
	}
	return e
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	outDir := filepath.Join(root, "spec", "investigation-v1", "conformance", "archetype-fixtures")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}

	tw := timeWindow{Start: "2026-06-27T09:00:00Z", End: "2026-06-27T09:30:00Z"}
	finish := expectFinish{State: "completed", ReportRequiredFields: []string{"executive_summary", "hypotheses", "confidence", "postmortem"}}
	start := expectStart{RequiredEvidenceMin: 1, PlanQuestionsMin: 1}

	defs := []fixture{
		{ArchetypeID: "deployment-failure", ScenarioID: "SC-ARCH-001", Tier: "core", Description: "Deployment precedes error spike; rollback restores service.",
			Start: startBlock{Question: "Why did checkout fail yesterday?", Service: "checkout-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "dep-1", Timestamp: "2026-06-27T09:01:00Z", Category: "deployment_events", Entity: "checkout-api", Summary: "Deployed checkout-api v2.4.0 to production"}},
				{{ID: "log-1", Timestamp: "2026-06-27T09:05:00Z", Category: "application_logs", Entity: "checkout-api", Summary: "HTTP 500 errors spiking on /checkout endpoint"},
					{ID: "alert-1", Timestamp: "2026-06-27T09:06:00Z", Category: "alert_events", Entity: "checkout-api", Summary: "Alert: checkout-api 5xx error rate exceeded 5%"}},
				{{ID: "metric-1", Timestamp: "2026-06-27T09:07:00Z", Category: "metrics", Entity: "checkout-api", Summary: "error_rate 12%, p99 latency 4s"},
					{ID: "dep-2", Timestamp: "2026-06-27T09:18:00Z", Category: "deployment_events", Entity: "checkout-api", Summary: "Rolled back checkout-api to v2.3.9; service recovered"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-deployment-caused", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "deployment-unrelated", ScenarioID: "SC-ARCH-002", Tier: "core", Description: "Symptoms begin before deployment lands.",
			Start: startBlock{Question: "Was the deployment the cause?", Service: "checkout-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "alert-1", Timestamp: "2026-06-27T09:01:00Z", Category: "alert_events", Entity: "checkout-api", Summary: "Alert: checkout-api 5xx error rate exceeded 5%"},
					{ID: "log-1", Timestamp: "2026-06-27T09:02:00Z", Category: "application_logs", Entity: "checkout-api", Summary: "HTTP 500 errors spiking on /checkout endpoint"}},
				{{ID: "dep-1", Timestamp: "2026-06-27T09:15:00Z", Category: "deployment_events", Entity: "checkout-api", Summary: "Deployed checkout-api v2.4.0 to production after incident began"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-deployment-unrelated", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "configuration-drift", ScenarioID: "SC-ARCH-003", Tier: "core", Description: "Configuration change caused service failure without Kubernetes lifecycle symptoms.",
			Start: startBlock{Question: "Why did the worker fail after the config change?", Service: "worker", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "cfg-1", Timestamp: "2026-06-27T09:00:00Z", Category: "configuration_changes", Entity: "worker", Summary: "Applied new ConfigMap with invalid DB_HOST env var"}},
				{{ID: "log-1", Timestamp: "2026-06-27T09:02:00Z", Category: "application_logs", Entity: "worker", Summary: "fatal: cannot connect using configured DB_HOST"},
					{ID: "alert-1", Timestamp: "2026-06-27T09:03:00Z", Category: "alert_events", Entity: "worker", Summary: "error rate spiked after ConfigMap change"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-configuration-change", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "infrastructure-failure", ScenarioID: "SC-ARCH-004", Tier: "core", Description: "Node or hypervisor failure.",
			Start: startBlock{Question: "Why did workloads fail?", Service: "api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "infra-1", Timestamp: "2026-06-27T09:01:00Z", Category: "infrastructure_events", Entity: "node-7", Summary: "Cloud event: node unavailable; hypervisor hardware alert"}},
				{{ID: "alert-1", Timestamp: "2026-06-27T09:02:00Z", Category: "alert_events", Entity: "api", Summary: "Alert: instance reboot required on host failure"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-infrastructure-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "database-saturation", ScenarioID: "SC-ARCH-005", Tier: "core", Description: "Database connection pool exhausted.",
			Start: startBlock{Question: "Why are orders timing out?", Service: "orders-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "log-1", Timestamp: "2026-06-27T09:03:00Z", Category: "application_logs", Entity: "orders-api", Summary: "database connection pool exhausted; query timeout"}},
				{{ID: "metric-1", Timestamp: "2026-06-27T09:04:00Z", Category: "metrics", Entity: "postgres-primary", Summary: "db connections 100/100 saturated, cpu 95%"},
					{ID: "db-1", Timestamp: "2026-06-27T09:05:00Z", Category: "database_events", Entity: "postgres-primary", Summary: "Postgres active connections at limit; write IOPS saturated"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-database-saturation", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "database-lock-contention", ScenarioID: "SC-ARCH-006", Tier: "core", Description: "Row lock queue with healthy database metrics.",
			Start: startBlock{Question: "Why did renameIdentityProvider writes spike to 66s p95?", Service: "auth-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{
				{{ID: "cfg-1", Timestamp: "2026-06-27T09:00:00Z", Category: "configuration_changes", Entity: "auth-api", Summary: "connection pool has no statement_timeout or lock_timeout configured"}},
				{{ID: "alert-1", Timestamp: "2026-06-27T09:05:00Z", Category: "alert_events", Entity: "auth-api", Summary: "p95 latency 65.8s vs 30s threshold"}},
				{{ID: "metric-1", Timestamp: "2026-06-27T09:05:30Z", Category: "metrics", Entity: "postgres-primary", Summary: "db connections 12/100, cpu 15%, reads fast"}},
				{
					{ID: "trace-1", Timestamp: "2026-06-27T09:10:00Z", Category: "trace_events", Entity: "auth-api", Summary: "write span 105.9s completed"},
					{ID: "trace-2", Timestamp: "2026-06-27T09:10:00Z", Category: "trace_events", Entity: "auth-api", Summary: "write span 26.7s completed"},
					{ID: "db-del", Timestamp: "2026-06-27T09:10:00Z", Category: "database_events", Entity: "identity_provider:pk-42", Summary: "DELETE on primary key row held lock 222.9s", Payload: map[string]any{"rows_affected": 1, "duration_ms": 222900}},
					{ID: "db-w1", Timestamp: "2026-06-27T09:10:00Z", Category: "database_events", Entity: "identity_provider:pk-42", Summary: "UPDATE queued behind lock", Payload: map[string]any{"rows_affected": 0, "duration_ms": 180000}},
					{ID: "db-w2", Timestamp: "2026-06-27T09:10:00Z", Category: "database_events", Entity: "identity_provider:pk-42", Summary: "UPDATE queued behind lock", Payload: map[string]any{"rows_affected": 0, "duration_ms": 150000}},
					{ID: "db-w3", Timestamp: "2026-06-27T09:10:00Z", Category: "database_events", Entity: "identity_provider:pk-42", Summary: "UPDATE queued behind lock", Payload: map[string]any{"rows_affected": 0, "duration_ms": 120000}},
				},
			},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-lock-contention", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "network-failure", ScenarioID: "SC-ARCH-007", Tier: "core", Description: "Routing or packet loss without DNS symptoms.",
			Start: startBlock{Question: "Why is connectivity failing?", Service: "payments-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "net-1", Timestamp: "2026-06-27T09:01:00Z", Category: "network_events", Entity: "payments-api", Summary: "network packet loss; tcp connection refused on private link"}},
				{{ID: "log-1", Timestamp: "2026-06-27T09:02:00Z", Category: "application_logs", Entity: "payments-api", Summary: "connection reset by peer; tcp socket unreachable"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-network-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "dns-failure", ScenarioID: "SC-ARCH-008", Tier: "core", Description: "DNS resolution failure.",
			Start: startBlock{Question: "Why can't payments reach the database?", Service: "payments-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "net-1", Timestamp: "2026-06-27T09:00:00Z", Category: "network_events", Entity: "payments-api", Summary: "DNS resolution failures for db.internal (NXDOMAIN)"}},
				{{ID: "log-1", Timestamp: "2026-06-27T09:01:00Z", Category: "application_logs", Entity: "payments-api", Summary: "could not resolve host db.internal; name resolution error"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-dns-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "certificate-tls-failure", ScenarioID: "SC-ARCH-009", Tier: "core", Description: "Expired TLS certificate.",
			Start: startBlock{Question: "Why is the API gateway rejecting requests?", Service: "api-gateway", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "sec-1", Timestamp: "2026-06-27T09:00:00Z", Category: "security_events", Entity: "api-gateway", Summary: "TLS certificate for api.example.com expired (x509)"}},
				{{ID: "log-1", Timestamp: "2026-06-27T09:01:00Z", Category: "application_logs", Entity: "api-gateway", Summary: "x509: certificate has expired; TLS handshake failure"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-certificate-expiry", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "auth-failure", ScenarioID: "SC-ARCH-010", Tier: "core", Description: "Authentication or authorization failure.",
			Start: startBlock{Question: "Why are users getting 401 errors?", Service: "auth-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "sec-1", Timestamp: "2026-06-27T09:01:00Z", Category: "security_events", Entity: "auth-api", Summary: "jwt token expired; authentication failed unauthorized 401"}},
				{{ID: "log-1", Timestamp: "2026-06-27T09:02:00Z", Category: "application_logs", Entity: "auth-api", Summary: "oauth validation failed; rbac denied access 403"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-auth-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "resource-exhaustion", ScenarioID: "SC-ARCH-011", Tier: "core", Description: "OOM and memory pressure.",
			Start: startBlock{Question: "Why does the image service keep restarting?", Service: "image-service", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "metric-1", Timestamp: "2026-06-27T09:00:00Z", Category: "metrics", Entity: "image-service", Summary: "memory usage climbing 60% -> 95%, heap growth over 2h"}},
				{{ID: "infra-1", Timestamp: "2026-06-27T09:30:00Z", Category: "infrastructure_events", Entity: "image-service", Summary: "Pod OOMKilled and restarted"},
					{ID: "log-1", Timestamp: "2026-06-27T09:31:00Z", Category: "application_logs", Entity: "image-service", Summary: "java.lang.OutOfMemoryError: Java heap space"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-resource-exhaustion", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "performance-regression", ScenarioID: "SC-ARCH-012", Tier: "core", Description: "Latency regression without database saturation.",
			Start: startBlock{Question: "Why did latency regress?", Service: "search-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "metric-1", Timestamp: "2026-06-27T09:01:00Z", Category: "metrics", Entity: "search-api", Summary: "latency regression: p99 slower than baseline, performance degraded"}},
				{{ID: "trace-1", Timestamp: "2026-06-27T09:02:00Z", Category: "trace_events", Entity: "search-api", Summary: "hot path regression on /search; performance degraded on read path"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-performance-regression", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "dependency-failure", ScenarioID: "SC-ARCH-013", Tier: "core", Description: "Downstream dependency timeout.",
			Start: startBlock{Question: "Why is checkout failing?", Service: "checkout-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "log-1", Timestamp: "2026-06-27T09:01:00Z", Category: "application_logs", Entity: "checkout-api", Summary: "downstream dependency inventory-service timeout"}},
				{{ID: "trace-1", Timestamp: "2026-06-27T09:02:00Z", Category: "trace_events", Entity: "checkout-api", Summary: "caller timeout waiting on inventory-service dependency"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-dependency-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "retry-storm", ScenarioID: "SC-ARCH-014", Tier: "core", Description: "Retry amplification.",
			Start: startBlock{Question: "Why did gateway latency explode?", Service: "gateway", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "log-1", Timestamp: "2026-06-27T09:00:00Z", Category: "application_logs", Entity: "gateway", Summary: "request timeout; retrying with backoff"}},
				{{ID: "metric-1", Timestamp: "2026-06-27T09:01:00Z", Category: "metrics", Entity: "gateway", Summary: "request rate 10x amplification consistent with a retry storm"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-retry-storm", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "kubernetes-failure", ScenarioID: "SC-ARCH-015", Tier: "core", Description: "Kubernetes scheduling and readiness faults.",
			Start: startBlock{Question: "Why are pods not ready?", Service: "worker", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "infra-1", Timestamp: "2026-06-27T09:01:00Z", Category: "infrastructure_events", Entity: "worker", Summary: "kubernetes pending pod; readiness probe failing on k8s workload"}},
				{{ID: "log-1", Timestamp: "2026-06-27T09:02:00Z", Category: "application_logs", Entity: "worker", Summary: "kubelet scheduler could not place pod; liveness probe failed"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-kubernetes-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "container-failure", ScenarioID: "SC-ARCH-016", Tier: "core", Description: "Container image pull failure.",
			Start: startBlock{Question: "Why won't the pod start?", Service: "api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "infra-1", Timestamp: "2026-06-27T09:01:00Z", Category: "infrastructure_events", Entity: "api", Summary: "docker image pull failed; registry unreachable"}},
				{{ID: "log-1", Timestamp: "2026-06-27T09:02:00Z", Category: "application_logs", Entity: "api", Summary: "oci runtime error: failed to pull image from containerd"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-container-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "storage-failure", ScenarioID: "SC-ARCH-017", Tier: "core", Description: "Storage volume failure.",
			Start: startBlock{Question: "Why are writes failing?", Service: "data-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "infra-1", Timestamp: "2026-06-27T09:01:00Z", Category: "infrastructure_events", Entity: "data-api", Summary: "ebs volume attach failed; pvc mount failed"}},
				{{ID: "metric-1", Timestamp: "2026-06-27T09:02:00Z", Category: "metrics", Entity: "data-api", Summary: "disk full; storage iops exhausted on san volume"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-storage-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "cache-failure", ScenarioID: "SC-ARCH-018", Tier: "core", Description: "Cache outage and stampede.",
			Start: startBlock{Question: "Why is origin load spiking?", Service: "catalog-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "log-1", Timestamp: "2026-06-27T09:01:00Z", Category: "application_logs", Entity: "catalog-api", Summary: "redis cache unavailable; cache stampede on hot keys"}},
				{{ID: "metric-1", Timestamp: "2026-06-27T09:02:00Z", Category: "metrics", Entity: "redis", Summary: "memcached hit ratio collapsed; cache eviction storm"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-cache-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "messaging-failure", ScenarioID: "SC-ARCH-019", Tier: "core", Description: "Message queue consumer lag.",
			Start: startBlock{Question: "Why is processing delayed?", Service: "orders-worker", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "log-1", Timestamp: "2026-06-27T09:01:00Z", Category: "application_logs", Entity: "orders-worker", Summary: "kafka consumer lag growing; rabbitmq queue buildup"}},
				{{ID: "metric-1", Timestamp: "2026-06-27T09:02:00Z", Category: "metrics", Entity: "orders-worker", Summary: "sqs message queue depth high; poison message detected on pulsar"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-messaging-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "load-balancer-failure", ScenarioID: "SC-ARCH-020", Tier: "core", Description: "Load balancer health check failure.",
			Start: startBlock{Question: "Why is traffic not reaching backends?", Service: "gateway", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "log-1", Timestamp: "2026-06-27T09:01:00Z", Category: "application_logs", Entity: "gateway", Summary: "nginx health check failed; alb backend unhealthy; envoy dead pool"}},
				{{ID: "log-2", Timestamp: "2026-06-27T09:02:00Z", Category: "application_logs", Entity: "gateway", Summary: "haproxy proxy failure: all backends marked unhealthy by load balancer"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-load-balancer-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "api-contract-failure", ScenarioID: "SC-ARCH-021", Tier: "core", Description: "Breaking API schema change.",
			Start: startBlock{Question: "Why are clients failing?", Service: "api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "log-1", Timestamp: "2026-06-27T09:01:00Z", Category: "application_logs", Entity: "api", Summary: "breaking change: schema change caused deserialization error"}},
				{{ID: "trace-1", Timestamp: "2026-06-27T09:02:00Z", Category: "trace_events", Entity: "api", Summary: "version mismatch between client and server payloads"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-api-contract-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "data-corruption", ScenarioID: "SC-ARCH-022", Tier: "core", Description: "Data integrity failure.",
			Start: startBlock{Question: "Why is data inconsistent?", Service: "ledger-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "log-1", Timestamp: "2026-06-27T09:01:00Z", Category: "application_logs", Entity: "ledger-api", Summary: "checksum mismatch detected; corrupted data on ledger shard"}},
				{{ID: "log-2", Timestamp: "2026-06-27T09:02:00Z", Category: "application_logs", Entity: "ledger-api", Summary: "partial write failed data integrity validation; checksum mismatch on commit"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-data-corruption", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "clock-failure", ScenarioID: "SC-ARCH-023", Tier: "core", Description: "Clock skew and NTP failure.",
			Start: startBlock{Question: "Why is auth failing intermittently?", Service: "auth-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "infra-1", Timestamp: "2026-06-27T09:01:00Z", Category: "infrastructure_events", Entity: "auth-api", Summary: "ntp sync lost; clock drift detected across nodes"}},
				{{ID: "sec-1", Timestamp: "2026-06-27T09:02:00Z", Category: "security_events", Entity: "auth-api", Summary: "token validation failed due to time skew on skewed clock"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-clock-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "feature-flag-failure", ScenarioID: "SC-ARCH-024", Tier: "core", Description: "Feature flag rollout mistake.",
			Start: startBlock{Question: "Why did behavior change for some users?", Service: "web-app", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "cfg-1", Timestamp: "2026-06-27T09:01:00Z", Category: "configuration_changes", Entity: "web-app", Summary: "feature flag rollout enabled via launchdarkly for wrong audience"}},
				{{ID: "log-1", Timestamp: "2026-06-27T09:02:00Z", Category: "application_logs", Entity: "web-app", Summary: "gradual rollout flag enabled caused regression for 50% traffic"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-feature-flag-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "security-incident", ScenarioID: "SC-ARCH-025", Tier: "core", Description: "Security breach detected.",
			Start: startBlock{Question: "Was this a security incident?", Service: "api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "sec-1", Timestamp: "2026-06-27T09:01:00Z", Category: "security_events", Entity: "api", Summary: "unauthorized access detected; credential compromise and exploit attempt"}},
				{{ID: "log-1", Timestamp: "2026-06-27T09:02:00Z", Category: "application_logs", Entity: "api", Summary: "intrusion attempt blocked; breach indicators in audit logs"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-security-incident", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "external-outage", ScenarioID: "SC-ARCH-026", Tier: "core", Description: "Third-party provider outage.",
			Start: startBlock{Question: "Why are payments failing?", Service: "payments-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "log-1", Timestamp: "2026-06-27T09:00:00Z", Category: "application_logs", Entity: "payments-api", Summary: "third-party payment provider outage; external service unavailable"}},
				{{ID: "log-2", Timestamp: "2026-06-27T09:01:00Z", Category: "application_logs", Entity: "payments-api", Summary: "vendor status page reports provider outage; SaaS external dependency down"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-external-outage", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "regional-failure", ScenarioID: "SC-ARCH-027", Tier: "core", Description: "Availability zone outage.",
			Start: startBlock{Question: "Why did us-east-1 fail?", Service: "api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "infra-1", Timestamp: "2026-06-27T09:01:00Z", Category: "infrastructure_events", Entity: "api", Summary: "availability zone failure in us-east-1; regional outage"}},
				{{ID: "net-1", Timestamp: "2026-06-27T09:02:00Z", Category: "network_events", Entity: "api", Summary: "single region dependency; zone failure isolated capacity"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-regional-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "dr-failover-failure", ScenarioID: "SC-ARCH-028", Tier: "core", Description: "Failover incomplete.",
			Start: startBlock{Question: "Why is DR not working?", Service: "db", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "infra-1", Timestamp: "2026-06-27T09:01:00Z", Category: "infrastructure_events", Entity: "db", Summary: "disaster recovery failover failed; recovery incomplete"}},
				{{ID: "infra-2", Timestamp: "2026-06-27T09:02:00Z", Category: "infrastructure_events", Entity: "db", Summary: "split brain detected after failover attempt"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-dr-failover-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "observability-failure", ScenarioID: "SC-ARCH-029", Tier: "core", Description: "Telemetry blind spot.",
			Start: startBlock{Question: "Why can't we see what happened?", Service: "api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "log-1", Timestamp: "2026-06-27T09:01:00Z", Category: "application_logs", Entity: "api", Summary: "missing metrics and missing logs for incident window; observability gap"}},
				{{ID: "log-2", Timestamp: "2026-06-27T09:02:00Z", Category: "application_logs", Entity: "api", Summary: "telemetry agent down; sampling dropped all traces; observability blind spot"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-observability-failure", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "human-error", ScenarioID: "SC-ARCH-030", Tier: "core", Description: "Manual operational mistake.",
			Start: startBlock{Question: "Was this caused by a human change?", Service: "api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "hum-1", Timestamp: "2026-06-27T09:01:00Z", Category: "human_context", Entity: "api", Summary: "operator ran manual change; human error during runbook execution"}},
				{{ID: "log-1", Timestamp: "2026-06-27T09:02:00Z", Category: "application_logs", Entity: "api", Summary: "wrong command executed; mistake in production shell"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-human-error", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "capacity-planning", ScenarioID: "SC-ARCH-031", Tier: "core", Description: "Traffic spike exceeded capacity.",
			Start: startBlock{Question: "Why did latency spike during the sale?", Service: "shop-api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "metric-1", Timestamp: "2026-06-27T09:01:00Z", Category: "metrics", Entity: "shop-api", Summary: "traffic spike 5x; autoscaling delayed; quota exceeded"}},
				{{ID: "infra-1", Timestamp: "2026-06-27T09:02:00Z", Category: "infrastructure_events", Entity: "shop-api", Summary: "capacity throttling under load"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-capacity-planning", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
		{ArchetypeID: "unknown-novel", ScenarioID: "SC-ARCH-032", Tier: "core", Description: "Insufficient evidence; unknown leads.",
			Start: startBlock{Question: "What caused the incident?", Service: "api", Goal: "root_cause", TimeWindow: tw},
			ExpectStart: start, ExpectFinish: finish,
			Batches: [][]evidence{{{ID: "alert-1", Timestamp: "2026-06-27T09:01:00Z", Category: "alert_events", Entity: "api", Summary: "Alert: elevated error rate"}}},
			ExpectAll: expectAll{LeadingHypothesisID: "hypothesis-unknown", MinConfidence: 10, MinHypotheses: 1, GraphNodesMin: 1}},
	}

	for i := range defs {
		defs[i].ExpectAll = enrichExpect(defs[i].ArchetypeID, defs[i].ExpectAll)
	}

	for _, fx := range defs {
		data, err := yaml.Marshal(fx)
		if err != nil {
			panic(err)
		}
		path := filepath.Join(outDir, fx.ArchetypeID+".yaml")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			panic(err)
		}
		fmt.Println("wrote", path)
	}
}
