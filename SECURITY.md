# Security Policy

## Reporting a vulnerability

Please report security issues through GitHub's private vulnerability reporting
("Report a vulnerability" on the Security tab of this repository). Do not open
a public issue for a vulnerability.

## Scope notes

gailwind reads source files and writes CSS. It does not execute, fetch or
render the files it scans, and it makes no network calls at runtime. The
embedded preflight is a verbatim copy of Tailwind v3.4.19's published CSS
output; see licenses/ for provenance.
