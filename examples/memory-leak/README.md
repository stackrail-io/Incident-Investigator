# Memory leak / OOM

**Service:** `image-service`  
**Question:** Why does the image service keep restarting?

Heap climbs 60% → 95% over two hours, then pod **OOMKilled** with `java.lang.OutOfMemoryError`.

Leading hypothesis: **`hypothesis-resource-exhaustion`**.
