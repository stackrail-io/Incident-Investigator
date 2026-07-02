//go:build ignore

// Regenerate example investigations: go run internal/spec/cmd/gen-examples/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	expkg "github.com/stackrail/incident-investigator/examples"
	"github.com/stackrail/incident-investigator/internal/spec"
)

type exampleDef struct {
	Dir       string
	FixtureID string
	BatchNames []string
}

var examples = []exampleDef{
	{Dir: "deployment-failure", FixtureID: "deployment-failure", BatchNames: []string{"01-deploy", "02-symptom-spike", "03-rollback-recovery"}},
	{Dir: "certificate-expiry", FixtureID: "certificate-tls-failure", BatchNames: []string{"01-tls-alert", "02-handshake-errors"}},
	{Dir: "dns-outage", FixtureID: "dns-failure", BatchNames: []string{"01-dns-failures", "02-app-errors"}},
	{Dir: "retry-storm", FixtureID: "retry-storm", BatchNames: []string{"01-retry-logs", "02-amplification-metrics"}},
	{Dir: "database-deadlock", FixtureID: "database-lock-contention", BatchNames: []string{"01-config-gap", "02-latency-alert", "03-db-metrics", "04-lock-queue"}},
	{Dir: "memory-leak", FixtureID: "resource-exhaustion", BatchNames: []string{"01-memory-climb", "02-oom-restart"}},
	{Dir: "regional-outage", FixtureID: "regional-failure", BatchNames: []string{"01-az-failure", "02-regional-impact"}},
}

func main() {
	root, err := findRepoRoot()
	if err != nil {
		panic(err)
	}
	for _, ex := range examples {
		if err := writeExample(root, ex); err != nil {
			panic(err)
		}
		fmt.Println("wrote", ex.Dir)
	}
}

func writeExample(root string, ex exampleDef) error {
	fxPath := filepath.Join(root, "spec/investigation-v1/conformance/archetype-fixtures", ex.FixtureID+".yaml")
	fx, err := spec.LoadConformanceFixture(fxPath)
	if err != nil {
		return err
	}
	outDir := filepath.Join(root, "examples", ex.Dir)
	if err := os.MkdirAll(filepath.Join(outDir, "evidence"), 0o755); err != nil {
		return err
	}
	inv := map[string]any{
		"description": fx.Description,
		"question":    fx.Start.Question,
		"service":     fx.Start.Service,
		"goal":        fx.Start.Goal,
		"time_window": fx.Start.TimeWindow,
	}
	if err := writeJSON(filepath.Join(outDir, "investigation.json"), inv); err != nil {
		return err
	}
	for i, batch := range fx.EvidenceBatches {
		name := fmt.Sprintf("%02d-batch", i+1)
		if i < len(ex.BatchNames) {
			name = ex.BatchNames[i]
		}
		items := make([]map[string]any, 0, len(batch))
		for _, e := range batch {
			item := map[string]any{
				"id": e.ID, "timestamp": e.Timestamp, "category": e.Category,
				"source": "datadog", "entity": e.Entity, "summary": e.Summary,
			}
			payload := e.Payload
			if payload == nil {
				payload = defaultPayload(ex.Dir, e.Category, e.Summary, e.ID)
			}
			if len(payload) > 0 {
				item["payload"] = payload
			}
			items = append(items, item)
		}
		batchDoc := map[string]any{
			"batch":       name,
			"description": batchDescription(ex.Dir, name),
			"evidence":    items,
		}
		if err := writeJSON(filepath.Join(outDir, "evidence", name+".json"), batchDoc); err != nil {
			return err
		}
	}
	return writeExpected(root, ex.Dir, fx)
}

func writeExpected(root, dir string, fx *spec.ConformanceFixture) error {
	outDir := filepath.Join(root, "examples", dir)
	findings := map[string]any{
		"leading_hypothesis_id": fx.ExpectAfterAllEvidence.LeadingHypothesisID,
		"min_confidence":        fx.ExpectAfterAllEvidence.MinConfidence,
		"competing_hypotheses_min": fx.ExpectAfterAllEvidence.MinHypotheses,
		"summary": fmt.Sprintf("After all evidence, %s should lead with confidence >= %.0f.",
			fx.ExpectAfterAllEvidence.LeadingHypothesisID, fx.ExpectAfterAllEvidence.MinConfidence),
	}
	questions := map[string]any{
		"min_plan_questions": fx.ExpectAfterStart.PlanQuestionsMin,
		"sample_open_questions": []string{
			"Did deployment happen before errors?",
			"Is the database healthy?",
			"Is the network path healthy?",
		},
	}
	graph := map[string]any{
		"min_nodes": fx.ExpectAfterAllEvidence.GraphNodesMin,
		"min_edges": 1,
		"expected_node_types": []string{"evidence", "hypothesis", "service"},
	}
	if err := writeJSON(filepath.Join(outDir, "expected-findings.json"), findings); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "expected-questions.json"), questions); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "expected-graph.json"), graph); err != nil {
		return err
	}
	report, err := expkg.RunReport(context.Background(), root, dir)
	if err != nil {
		return fmt.Errorf("run example %s: %w", dir, err)
	}
	return os.WriteFile(filepath.Join(outDir, "expected-report.md"), []byte(report.Postmortem), 0o644)
}

func batchDescription(dir, name string) string {
	descs := map[string]string{
		"deployment-failure:01-deploy":            "Production rollout of checkout-api v2.4.0 via CI pipeline.",
		"deployment-failure:02-symptom-spike":     "5xx errors and paging alerts immediately after deploy.",
		"deployment-failure:03-rollback-recovery": "Rollback to v2.3.9; error rate returns to baseline.",
		"certificate-expiry:01-tls-alert":         "Certificate monitor fires on api.example.com expiry.",
		"certificate-expiry:02-handshake-errors":    "Application logs show TLS handshake failures.",
		"dns-outage:01-dns-failures":              "CoreDNS / resolver NXDOMAIN for db.internal.",
		"dns-outage:02-app-errors":                "payments-api connection pool errors referencing DNS.",
		"retry-storm:01-retry-logs":               "Gateway logs show cascading timeouts and retries.",
		"retry-storm:02-amplification-metrics":    "Request rate spike consistent with retry storm.",
		"database-deadlock:01-config-gap":         "Pool config missing lock_timeout / statement_timeout.",
		"database-deadlock:02-latency-alert":      "p95 write latency breaches SLO; DB metrics still healthy.",
		"database-deadlock:03-db-metrics":           "Postgres CPU and connection headroom remain healthy.",
		"database-deadlock:04-lock-queue":         "Traces and pg_locks show queued writers behind long DELETE.",
		"memory-leak:01-memory-climb":             "Heap growth over two hours preceding incident.",
		"memory-leak:02-oom-restart":              "Pod OOMKilled; JVM heap space error in logs.",
		"regional-outage:01-az-failure":           "us-east-1 AZ impairment reported by cloud provider.",
		"regional-outage:02-regional-impact":      "Single-region capacity loss; cross-AZ traffic shift incomplete.",
	}
	if d, ok := descs[dir+":"+name]; ok {
		return d
	}
	return "Evidence batch " + name
}

func defaultPayload(dir, category, summary, id string) map[string]any {
	switch category {
	case "deployment_events":
		if strings.Contains(strings.ToLower(summary), "rollback") {
			return map[string]any{
				"environment": "production", "version": "v2.3.9", "action": "rollback",
				"previous_version": "v2.4.0", "deploy_id": "dep-checkout-rollback-9912",
				"triggered_by": "oncall", "new_deploy": false,
			}
		}
		return map[string]any{
			"environment": "production", "version": "v2.4.0", "deploy_id": "dep-checkout-8842",
			"pipeline": "github-actions", "commit": "a1b2c3d", "new_deploy": true,
		}
	case "application_logs":
		p := map[string]any{"service": entityFromSummary(summary), "level": "error"}
		if strings.Contains(summary, "500") {
			p["http_status"] = 500
			p["error_count_5m"] = 1240
			p["endpoint"] = "/checkout"
		}
		if strings.Contains(strings.ToLower(summary), "oom") || strings.Contains(strings.ToLower(summary), "outofmemory") {
			p["exception"] = "java.lang.OutOfMemoryError"
			p["heap_used_mb"] = 4096
		}
		if strings.Contains(strings.ToLower(summary), "x509") || strings.Contains(strings.ToLower(summary), "certificate") {
			p["tls_error"] = "certificate has expired"
			p["sni"] = "api.example.com"
		}
		if strings.Contains(strings.ToLower(summary), "dns") || strings.Contains(strings.ToLower(summary), "resolve") {
			p["host"] = "db.internal"
			p["resolver_error"] = "NXDOMAIN"
		}
		if strings.Contains(strings.ToLower(summary), "retry") {
			p["retry_count"] = 47
			p["upstream"] = "payments-api"
		}
		return p
	case "alert_events":
		return map[string]any{
			"monitor": id, "severity": "critical", "status": "firing",
			"threshold": "5%", "current_value": "12%",
		}
	case "metrics":
		p := map[string]any{"source": "datadog"}
		if strings.Contains(summary, "error_rate") {
			p["error_rate_pct"] = 12
			p["p99_latency_ms"] = 4000
		}
		if strings.Contains(summary, "amplification") || strings.Contains(summary, "10x") {
			p["request_rate_rps"] = 8500
			p["baseline_rps"] = 850
		}
		if strings.Contains(summary, "memory") {
			p["memory_pct"] = 95
			p["heap_growth_mb_per_hour"] = 180
		}
		if strings.Contains(strings.ToLower(summary), "connection") {
			p["connections_active"] = 12
			p["connections_max"] = 100
			p["cpu_pct"] = 15
		}
		return p
	case "security_events":
		return map[string]any{
			"cert_cn": "api.example.com", "days_expired": 3,
			"issuer": "Let's Encrypt", "not_after": "2026-06-24T00:00:00Z",
		}
	case "network_events":
		p := map[string]any{"source": "vpc-flow"}
		if strings.Contains(strings.ToLower(summary), "dns") {
			p["query"] = "db.internal"
			p["rcode"] = "NXDOMAIN"
		}
		if strings.Contains(strings.ToLower(summary), "zone") || strings.Contains(strings.ToLower(summary), "az") {
			p["region"] = "us-east-1"
			p["failed_az"] = "us-east-1a"
		}
		return p
	case "infrastructure_events":
		if strings.Contains(strings.ToLower(summary), "oom") {
			return map[string]any{"reason": "OOMKilled", "exit_code": 137, "restart_count": 4}
		}
		return map[string]any{"region": "us-east-1", "event_type": "AZImpaired", "provider_status": "major"}
	case "configuration_changes":
		return map[string]any{
			"component": "connection_pool", "statement_timeout_ms": nil, "lock_timeout_ms": nil,
			"changed_by": "helm-upgrade", "config_changed": true,
		}
	case "trace_events":
		return map[string]any{"operation": "renameIdentityProvider", "span_kind": "server"}
	default:
		return nil
	}
}

func entityFromSummary(summary string) string {
	if i := strings.Index(summary, " "); i > 0 {
		return ""
	}
	return ""
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
