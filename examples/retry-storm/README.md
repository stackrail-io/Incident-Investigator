# Retry storm

**Service:** `gateway`  
**Question:** Why did gateway latency explode?

Upstream timeouts trigger aggressive retries; request rate amplifies ~10× baseline — classic retry storm pattern.

Leading hypothesis: **`hypothesis-retry-storm`**.
