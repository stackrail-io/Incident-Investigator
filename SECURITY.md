# Security Policy

## Supported versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

If you believe you have found a security issue in Incident Investigator, report it
responsibly using one of these channels:

1. **GitHub private security advisories** (preferred)  
   [Create a private security advisory](https://github.com/stackrail/incident-investigator/security/advisories/new)

2. **Email**  
   Send details to **security@stackrail.io** with the subject line  
   `Incident Investigator Security Report`.

Include as much of the following as you can:

- Description of the issue and potential impact
- Steps to reproduce or a proof of concept
- Affected version(s)
- Any suggested fix or mitigation

## What to expect

- **Acknowledgment** within 3 business days
- **Initial assessment** within 7 business days
- We will work with you to understand and validate the report
- We will coordinate disclosure and credit (if desired) once a fix is available

## Scope

In scope:

- The Incident Investigator MCP server and investigation runtime
- Denial of service, memory exhaustion, or crash issues in session handling
- Issues that could leak investigation session data across tenants (when deployed in shared environments)

Out of scope:

- Vulnerabilities in MCP clients (Cursor, Claude Code, etc.) — report those to the client vendor
- Misconfiguration of MCP clients or exposing the server to untrusted networks without access controls
- Issues in third-party dependencies — we will still triage and upgrade dependencies as needed

## Secure deployment notes

Incident Investigator is designed as a **local MCP server over stdio**. It stores
investigation state **in memory only** and has **no authentication** in v0.1.x.

Do not expose the process to untrusted users or networks without appropriate
isolation. Treat submitted evidence as sensitive operational data.

## Bug reports vs. security reports

- **Bugs** (incorrect hypotheses, planner behavior, crashes with benign input): use the
  [bug report template](https://github.com/stackrail/incident-investigator/issues/new?template=bug_report.yml)
- **Security issues**: use this policy only
