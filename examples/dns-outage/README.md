# DNS outage

**Service:** `payments-api`  
**Question:** Why can't payments reach the database?

Resolver returns **NXDOMAIN** for `db.internal`. Application logs show name resolution failures distinct from generic packet loss.

Leading hypothesis: **`hypothesis-dns-failure`**.
