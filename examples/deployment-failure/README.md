# Deployment failure

**Service:** `checkout-api`  
**Question:** Why did checkout fail yesterday?

## Timeline

| Time (UTC) | Event |
| ---------- | ----- |
| 09:01 | `checkout-api` v2.4.0 deployed to production (`dep-1`) |
| 09:05–09:07 | HTTP 500 spike on `/checkout`, error rate 12%, p99 4s |
| 09:18 | Rollback to v2.3.9; service recovers |

## Evidence batches

1. `evidence/01-deploy.json` — CI deploy event with version and commit metadata
2. `evidence/02-symptom-spike.json` — application logs, paging alert, error metrics
3. `evidence/03-rollback-recovery.json` — rollback deployment and recovery signal

## Expected outcome

Leading hypothesis: **`hypothesis-deployment-caused`** — deploy preceded symptoms; rollback correlated with recovery.

```bash
go test ./examples/ -run Deployment
```
