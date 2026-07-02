# Expected reasoning path

1. **Temporal** — deployment timestamp precedes first 5xx log line.
2. **deploy-before-errors** — confirmed when deploy + symptom evidence present.
3. **Hypothesis field** — deployment-caused leads; deployment-unrelated trails.
4. **Rollback** — recovery evidence increases confidence in deploy causality.
