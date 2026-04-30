# Agent Instructions

## Pre-Push Checklist

Before pushing any changes, always run:

```bash
make pre-push
```

This will:
1. Run Go formatting (`gofmt`)
2. Run Go tests
3. Run Playwright tests

## Individual Commands

```bash
make test           # Run all tests (go + playwright)
make test-go        # Run Go tests only
make test-playwright # Run Playwright tests only
make fmt            # Run Go formatting only
make check          # Run tests only (no formatting)
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

### Single test file
```bash
go test -v -run TestName ./...
```

### Single playwright test
```bash
npx playwright test tests/filename.spec.ts
```