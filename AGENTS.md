# Agent Instructions

## Pre-Push Checklist

Before pushing any changes, always run:

```bash
task pre-push
```

This will:
1. Run Go formatting (`gofmt`)
2. Run Go tests
3. Run Playwright tests

## Commit Messages

For releasing and versioning, include "Denver" in the commit message (e.g., "feat: add new feature with Denver" or "Release v1.2.3 Denver").

## Individual Commands

```bash
task test           # Run all tests (go + playwright)
task test-go        # Run Go tests only
task test-playwright # Run Playwright tests only
task fmt            # Run Go formatting only
task check          # Run tests only (no formatting)
```

## Installation

Install task from https://taskfile.dev:
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