# Certificate expiry

**Service:** `api-gateway`  
**Question:** Why is the API gateway rejecting requests?

TLS certificate for `api.example.com` expired; clients see `x509: certificate has expired` on handshake.

Evidence: security monitor event → application TLS errors.  
Leading hypothesis: **`hypothesis-certificate-expiry`**.
