# Expected reasoning path

1. **database-healthy** — confirmed; saturation ruled out.
2. **Lock signals** — holder/waiter queue detected from database_events payloads.
3. **lock-contention-queue** — confirmed from trace + lock duration evidence.
