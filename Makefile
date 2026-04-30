.PHONY: test test-go test-playwright fmt check pre-push

test: test-go test-playwright
	@echo "All tests passed!"

test-go:
	@echo "Running Go tests..."
	go test ./...

test-playwright:
	@echo "Running Playwright tests..."
	npx playwright test

fmt:
	@echo "Running Go formatting..."
	gofmt -w .

check: test-go test-playwright
	@echo "All checks passed!"

pre-push: fmt check
	@echo "Ready to push!"