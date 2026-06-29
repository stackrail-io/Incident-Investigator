// Package fixtures provides realistic, vendor-neutral incident scenarios used to
// exercise the planner, investigation graph, timeline, hypotheses and confidence
// engines. Each fixture submits evidence in batches to mimic an assistant
// progressively gathering data.
package fixtures

import (
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
)

// Fixture is a complete, replayable incident scenario.
type Fixture struct {
	Name     string
	Question string
	Service  string
	Window   model.TimeWindow
	// Batches are submitted in order, simulating incremental evidence gathering.
	Batches [][]*model.Evidence
	// ExpectLeading is the hypothesis id expected to lead after all batches.
	ExpectLeading string
}

// base is the reference time all fixtures are anchored to.
var base = time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)

func at(min, sec int) time.Time {
	return base.Add(time.Duration(min)*time.Minute + time.Duration(sec)*time.Second)
}

func ev(id string, t time.Time, cat model.Category, entity, summary string, payload map[string]any) *model.Evidence {
	return &model.Evidence{
		ID:        id,
		Timestamp: t,
		Category:  cat,
		Source:    "provided_by_client",
		Entity:    entity,
		Summary:   summary,
		Payload:   payload,
	}
}

// All returns every built-in fixture.
func All() []Fixture {
	return []Fixture{
		BadDeployment(),
		DatabaseOutage(),
		CertificateExpiry(),
		DNSOutage(),
		KubernetesRestartLoop(),
		MemoryLeak(),
		RetryStorm(),
		LockContention(),
		DependencyFailure(),
		ExternalOutage(),
	}
}

// BadDeployment: a release immediately precedes an error spike, then a rollback
// restores service. The canonical "deployment caused it" signature.
func BadDeployment() Fixture {
	return Fixture{
		Name:          "bad_deployment",
		Question:      "Why did checkout fail yesterday?",
		Service:       "checkout-api",
		Window:        model.TimeWindow{Start: at(0, 0), End: at(30, 0)},
		ExpectLeading: "hypothesis-deployment-caused",
		Batches: [][]*model.Evidence{
			{
				ev("dep-1", at(1, 0), model.CategoryDeploymentEvents, "checkout-api",
					"Deployed checkout-api v2.4.0 to production", map[string]any{"region": "us-east-1", "version": "v2.4.0"}),
			},
			{
				ev("log-1", at(5, 0), model.CategoryApplicationLogs, "checkout-api",
					"HTTP 500 errors spiking on /checkout endpoint", map[string]any{"api": "/checkout", "customer": "all-web"}),
				ev("alert-1", at(6, 0), model.CategoryAlertEvents, "checkout-api",
					"Alert: checkout-api 5xx error rate exceeded 5%", map[string]any{"region": "us-east-1"}),
			},
			{
				ev("metric-1", at(7, 0), model.CategoryMetrics, "checkout-api",
					"error_rate 12%, p99 latency 4s", map[string]any{"region": "us-east-1"}),
				ev("dep-2", at(18, 0), model.CategoryDeploymentEvents, "checkout-api",
					"Rolled back checkout-api to v2.3.9; service recovered", map[string]any{"version": "v2.3.9"}),
			},
		},
	}
}

// DatabaseOutage: connection-pool exhaustion and a primary failover degrade an
// orders service. No deployment is involved.
func DatabaseOutage() Fixture {
	return Fixture{
		Name:          "database_outage",
		Question:      "Why are orders timing out?",
		Service:       "orders-api",
		Window:        model.TimeWindow{Start: at(0, 0), End: at(20, 0)},
		ExpectLeading: "hypothesis-database-saturation",
		Batches: [][]*model.Evidence{
			{
				ev("log-1", at(3, 0), model.CategoryApplicationLogs, "orders-api",
					"database connection pool exhausted; query timeout", map[string]any{"api": "/orders"}),
				ev("alert-1", at(3, 30), model.CategoryAlertEvents, "orders-api",
					"Alert: database connections saturated", nil),
			},
			{
				ev("metric-1", at(4, 0), model.CategoryMetrics, "postgres-primary",
					"db connections 100/100 saturated, cpu 95%", map[string]any{"region": "us-west-2"}),
				ev("db-1", at(5, 0), model.CategoryDatabaseEvents, "postgres-primary",
					"Postgres primary failover triggered; replica lag high", map[string]any{"region": "us-west-2"}),
			},
		},
	}
}

// CertificateExpiry: an expired TLS certificate breaks all secure connections.
func CertificateExpiry() Fixture {
	return Fixture{
		Name:          "certificate_expiry",
		Question:      "Why is the API gateway rejecting all requests?",
		Service:       "api-gateway",
		Window:        model.TimeWindow{Start: at(0, 0), End: at(15, 0)},
		ExpectLeading: "hypothesis-certificate-expiry",
		Batches: [][]*model.Evidence{
			{
				ev("sec-1", at(0, 0), model.CategorySecurityEvents, "api-gateway",
					"TLS certificate for api.example.com expired (x509)", map[string]any{"api": "*"}),
			},
			{
				ev("log-1", at(1, 0), model.CategoryApplicationLogs, "api-gateway",
					"x509: certificate has expired; TLS handshake failure", nil),
				ev("alert-1", at(2, 0), model.CategoryAlertEvents, "api-gateway",
					"Alert: TLS handshake error rate 100%", nil),
			},
		},
	}
}

// DNSOutage: name resolution failures sever connectivity to a dependency.
func DNSOutage() Fixture {
	return Fixture{
		Name:          "dns_outage",
		Question:      "Why can't payments reach the database?",
		Service:       "payments-api",
		Window:        model.TimeWindow{Start: at(0, 0), End: at(15, 0)},
		ExpectLeading: "hypothesis-network-dns",
		Batches: [][]*model.Evidence{
			{
				ev("net-1", at(0, 0), model.CategoryNetworkEvents, "payments-api",
					"DNS resolution failures for db.internal (NXDOMAIN)", nil),
			},
			{
				ev("log-1", at(1, 0), model.CategoryApplicationLogs, "payments-api",
					"could not resolve host db.internal; name resolution error", nil),
				ev("alert-1", at(2, 0), model.CategoryAlertEvents, "payments-api",
					"Alert: payments-api 503 errors", map[string]any{"api": "/pay"}),
			},
		},
	}
}

// KubernetesRestartLoop: a bad configuration change sends pods into a crash loop.
func KubernetesRestartLoop() Fixture {
	return Fixture{
		Name:          "kubernetes_restart_loop",
		Question:      "Why is the worker pod crash looping?",
		Service:       "worker",
		Window:        model.TimeWindow{Start: at(0, 0), End: at(15, 0)},
		ExpectLeading: "hypothesis-configuration-change",
		Batches: [][]*model.Evidence{
			{
				ev("cfg-1", at(0, 0), model.CategoryConfigurationChanges, "worker",
					"Applied new ConfigMap with invalid DB_HOST env var", map[string]any{"change": "configmap"}),
			},
			{
				ev("infra-1", at(2, 0), model.CategoryInfrastructureEvents, "worker",
					"Pod worker-xyz CrashLoopBackOff, restart count 9", nil),
				ev("log-1", at(3, 0), model.CategoryApplicationLogs, "worker",
					"fatal: cannot connect using configured DB_HOST", nil),
				ev("alert-1", at(4, 0), model.CategoryAlertEvents, "worker",
					"Alert: worker availability 0%", nil),
			},
		},
	}
}

// MemoryLeak: gradual heap growth ends in OOM kills and restarts.
func MemoryLeak() Fixture {
	return Fixture{
		Name:          "memory_leak",
		Question:      "Why does the image service keep restarting?",
		Service:       "image-service",
		Window:        model.TimeWindow{Start: at(0, 0), End: at(40, 0)},
		ExpectLeading: "hypothesis-resource-exhaustion",
		Batches: [][]*model.Evidence{
			{
				ev("metric-1", at(0, 0), model.CategoryMetrics, "image-service",
					"memory usage climbing 60% -> 95%, heap growth over 2h", map[string]any{"region": "eu-west-1"}),
			},
			{
				ev("infra-1", at(30, 0), model.CategoryInfrastructureEvents, "image-service",
					"Pod OOMKilled and restarted", nil),
				ev("log-1", at(31, 0), model.CategoryApplicationLogs, "image-service",
					"java.lang.OutOfMemoryError: Java heap space", nil),
				ev("alert-1", at(32, 0), model.CategoryAlertEvents, "image-service",
					"Alert: image-service OOM restarts", nil),
			},
		},
	}
}

// RetryStorm: a transient fault is amplified by aggressive client retries.
func RetryStorm() Fixture {
	return Fixture{
		Name:          "retry_storm",
		Question:      "Why did the gateway latency explode?",
		Service:       "gateway",
		Window:        model.TimeWindow{Start: at(0, 0), End: at(15, 0)},
		ExpectLeading: "hypothesis-retry-storm",
		Batches: [][]*model.Evidence{
			{
				ev("log-1", at(0, 0), model.CategoryApplicationLogs, "gateway",
					"request timeout; retrying with backoff", nil),
			},
			{
				ev("metric-1", at(1, 0), model.CategoryMetrics, "gateway",
					"request rate 10x amplification consistent with a retry storm", nil),
				ev("alert-1", at(2, 0), model.CategoryAlertEvents, "gateway",
					"Alert: gateway p99 latency 30s", map[string]any{"api": "/v1"}),
				ev("trace-1", at(3, 0), model.CategoryTraceEvents, "gateway",
					"retries cascading to inventory-service", nil),
			},
		},
	}
}

// LockContention: a long-held row lock blocks concurrent writers on the same
// primary key; missing lock timeouts amplify wait time. Database capacity is
// healthy — this is not saturation.
func LockContention() Fixture {
	release := at(10, 0)
	return Fixture{
		Name:          "lock_contention",
		Question:      "Why did renameIdentityProvider writes spike to 66s p95?",
		Service:       "auth-api",
		Window:        model.TimeWindow{Start: at(0, 0), End: at(20, 0)},
		ExpectLeading: "hypothesis-lock-contention",
		Batches: [][]*model.Evidence{
			{
				ev("cfg-1", at(0, 0), model.CategoryConfigurationChanges, "auth-api",
					"connection pool has no statement_timeout or lock_timeout configured", nil),
			},
			{
				ev("alert-1", at(5, 0), model.CategoryAlertEvents, "auth-api",
					"p95 latency 65.8s vs 30s threshold", map[string]any{"p95_ms": 65800}),
				ev("metric-1", at(5, 30), model.CategoryMetrics, "postgres-primary",
					"db connections 12/100, cpu 15%, reads fast", map[string]any{"connections": "12/100"}),
			},
			{
				ev("trace-1", release, model.CategoryTraceEvents, "auth-api",
					"write span 105.9s completed", map[string]any{"duration_ms": 105900}),
				ev("trace-2", release, model.CategoryTraceEvents, "auth-api",
					"write span 26.7s completed", map[string]any{"duration_ms": 26700}),
				ev("db-del", release, model.CategoryDatabaseEvents, "identity_provider:pk-42",
					"DELETE on primary key row held lock 222.9s", map[string]any{"rows_affected": 1, "duration_ms": 222900}),
				ev("db-w1", release, model.CategoryDatabaseEvents, "identity_provider:pk-42",
					"UPDATE queued behind lock", map[string]any{"rows_affected": 0, "duration_ms": 180000}),
				ev("db-w2", release, model.CategoryDatabaseEvents, "identity_provider:pk-42",
					"UPDATE queued behind lock", map[string]any{"rows_affected": 0, "duration_ms": 150000}),
				ev("db-w3", release, model.CategoryDatabaseEvents, "identity_provider:pk-42",
					"UPDATE queued behind lock", map[string]any{"rows_affected": 0, "duration_ms": 120000}),
			},
		},
	}
}

// DependencyFailure: a downstream inventory service times out and degrades the gateway.
func DependencyFailure() Fixture {
	return Fixture{
		Name:          "dependency_failure",
		Question:      "Why is checkout failing with 503 errors?",
		Service:       "checkout-api",
		Window:        model.TimeWindow{Start: at(0, 0), End: at(15, 0)},
		ExpectLeading: "hypothesis-dependency-failure",
		Batches: [][]*model.Evidence{
			{
				ev("log-1", at(1, 0), model.CategoryApplicationLogs, "checkout-api",
					"downstream dependency inventory-service timeout", nil),
			},
			{
				ev("trace-1", at(2, 0), model.CategoryTraceEvents, "checkout-api",
					"caller timeout waiting on inventory-service dependency", nil),
				ev("alert-1", at(3, 0), model.CategoryAlertEvents, "checkout-api",
					"Alert: checkout-api 503 errors", map[string]any{"api": "/checkout"}),
			},
		},
	}
}

// ExternalOutage: a third-party payment provider outage causes internal failures.
func ExternalOutage() Fixture {
	return Fixture{
		Name:          "external_outage",
		Question:      "Why are payments failing across all regions?",
		Service:       "payments-api",
		Window:        model.TimeWindow{Start: at(0, 0), End: at(20, 0)},
		ExpectLeading: "hypothesis-external-outage",
		Batches: [][]*model.Evidence{
			{
				ev("log-1", at(0, 0), model.CategoryApplicationLogs, "payments-api",
					"third-party payment provider outage; external service unavailable", nil),
			},
			{
				ev("net-1", at(1, 0), model.CategoryNetworkEvents, "payments-api",
					"vendor status page reports provider outage in us-east-1", nil),
				ev("alert-1", at(2, 0), model.CategoryAlertEvents, "payments-api",
					"Alert: payment API failures 100%", nil),
			},
		},
	}
}
