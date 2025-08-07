
# Kubewatch TUI - Agent Guidelines

## Build/Test Commands
- **Build:** `make build` (current OS/arch) or `make build-all` (all platforms)
- **Test all:** `make test` or `go test -v -race ./...`
- **Test single:** `go test -v ./internal/ui/views -run TestResourceView`
- **Test package:** `go test -v ./internal/ui/...`
- **Lint:** `make lint` (uses golangci-lint)
- **Format:** `make fmt` (gofmt -s -w + go mod tidy)
- **Coverage:** `make coverage` (generates coverage.html)

## Code Style
- **Imports:** Group stdlib, external deps, internal packages (separated by blank lines)
- **Error handling:** Always check errors, wrap with context using fmt.Errorf
- **Naming:** Use camelCase for vars/funcs, PascalCase for exported types/funcs
- **Testing:** Table-driven tests preferred, use t.Run for subtests
- **Context:** Pass context.Context as first parameter, respect cancellation
- **Channels:** Always close channels when done, use select for non-blocking ops
- **Interfaces:** Define at consumer side, keep minimal
- **Comments:** Add for exported types/funcs, avoid obvious comments

## Specialized Agents
Use these agents for domain-specific tasks:
- **frontend-tui:** Bubble Tea UI, Lipgloss styling, terminal layouts (internal/ui/)
- **backend-k8s:** Kubernetes client-go, informers, watch streams (internal/k8s/)
- **platform-infra:** Build system, Makefile, CI/CD, testing infrastructure
- **test-quality-expert:** Unit tests, integration tests, test coverage, best practices
- **project-completion-validator:** Verify all tests pass, features complete, builds succeed

## Testing Best Practices
- **Coverage Goal:** Maintain >80% overall, >90% for UI package
- **Test Structure:** Use table-driven tests with descriptive names
- **Mock Generation:** Use mockgen for interfaces: `//go:generate mockgen`
- **Test Isolation:** Each test creates own app instance, no external dependencies
- **Key Testing:** Test all key bindings, mode transitions, error paths
- **Performance:** Keep tests fast (<5s), use minimal setup
- **Integration Tests:** Test component interactions, use test helpers in test_helpers.go
- **Before Completion:** Always run `make test` and `make lint` before marking done

## Architecture Notes
- **Module Boundaries:** internal/k8s (backend), internal/ui (frontend), internal/core (shared)
- **Event System:** Use channels for communication between modules
- **Performance Targets:** <2s startup, <50MB memory, handle 500+ resources smoothly
- **Real-time Updates:** Use informers not polling, handle reconnections gracefully

## Efficiency & Parallelism
- **Parallel Execution:** Use goroutines for concurrent operations (log streaming, resource watching)
- **Batch Operations:** Process multiple resources simultaneously, avoid sequential loops
- **Non-blocking UI:** Never block the UI thread, use tea.Cmd for async operations
- **Concurrent Testing:** Run test packages in parallel with `go test -parallel`
- **Agent Coordination:** Launch multiple specialized agents concurrently when tasks are independent
- **Resource Watching:** Use separate goroutines for each resource type's informer
- **Log Aggregation:** Stream multiple pod logs concurrently with proper synchronization

## Project Completion
Consult Project Completion Validator subagent before marking tasks complete. It is never acceptable to mark something complete with a failing test, not for any reason.