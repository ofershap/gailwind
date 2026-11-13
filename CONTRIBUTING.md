# Contributing

Issues and pull requests are welcome.

## Setup

Go 1.26 or newer. No other dependencies for the library, CLI and tests.
`parity/` needs Python plus the official `tailwindcss@3.4.19` CLI (used to
generate reference output) and is not part of the normal test run.

## Checks

Run everything CI runs before opening a PR:

```bash
go vet ./...
gofmt -l .
go test -race ./...
golangci-lint run ./...
```

## Rules

- Every compiled class must match what tailwindcss v3.4.19 emits. If you add
  or change a rule, extend the corpus and re-run the parity check; do not
  hand-write "close enough" CSS.
- Keep the public API small. A new entry point needs a real project that
  needs it.
- Claims in the README must match what the tests demonstrate.
