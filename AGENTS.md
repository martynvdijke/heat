# Agent Instructions

## Pre-Push Checklist

Before pushing any changes, always run:

```bash
task pre-push
```

This will:

1. Run Go formatting (`gofmt`)
2. Run Go tests
3. Run Go vet + vulnerability scan (`govulncheck`)
4. Compile TypeScript
5. Build the Go binary

## Commit Messages

Use conventional commits with semver (e.g., "feat: add new feature", "fix: resolve crash", "chore: bump deps").

## Individual Commands

```bash
task test           # Run Go unit tests only
task test:e2e       # Run Playwright end-to-end tests only
task fmt            # Run Go formatting only
task vuln           # Run Go vet + govulncheck vulnerability scan
task build          # Build the Go binary (compiles TS first)
task check          # Run tests only (no formatting)
```

## Installation

Install task from <https://taskfile.dev>:

```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

## Running Tests Directly

### Go tests

```bash
go test ./...
```

### Playwright tests

```bash
npx playwright test
```
