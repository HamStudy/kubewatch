
# Kubewatch TUI - Agent Guidelines

## Build/Test Commands
- **Build:** `make build` (current OS/arch) or `make build-all` (all platforms)
- **Test all:** `make test` or `go test -v -race ./...` **← MANDATORY before task completion**
- **Test single:** `go test -v ./internal/ui/views -run TestResourceView`
- **Test package:** `go test -v ./internal/ui/...`
- **Lint:** `make lint` (uses golangci-lint)
- **Format:** `make fmt` (gofmt -s -w + go mod tidy)
- **Coverage:** `make coverage` (generates coverage.html)

## ⚠️ MANDATORY TEST VALIDATION ⚠️
**BEFORE CLAIMING ANY TASK COMPLETE:**
1. **ALWAYS run `make test` (full test suite) - NO EXCEPTIONS**
2. **NEVER run partial tests and claim "all tests pass"**
3. **Run `go test -v -race ./...` to verify ALL tests pass**
4. **Project Completion Validator MUST verify `make test` passes before sign-off**
5. **ALL tests must pass - no justifications for failures accepted**

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
- **project-completion-validator:** **MANDATORY** - Verify `make test` passes, all features complete, builds succeed

## Testing Best Practices
- **Coverage Goal:** Maintain >80% overall, >90% for UI package
- **Test Structure:** Use table-driven tests with descriptive names
- **Mock Generation:** Use mockgen for interfaces: `//go:generate mockgen`
- **Test Isolation:** Each test creates own app instance, no external dependencies
- **Key Testing:** Test all key bindings, mode transitions, error paths
- **Performance:** Keep tests fast (<5s), use minimal setup
- **Integration Tests:** Test component interactions, use test helpers in test_helpers.go
- **MANDATORY VALIDATION:** Run `make test` (full suite) before ANY task completion - NO PARTIAL TESTS

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


## 🚨 CRITICAL RULES - ZERO TOLERANCE 🚨

### MANDATORY TEST VALIDATION - NO EXCEPTIONS
1. **ALWAYS run `make test` (full test suite) before claiming ANY task complete**
2. **NEVER run partial tests and claim "all tests pass"** - this is FORBIDDEN
3. **ALL tests must pass - ZERO TOLERANCE for failing tests**
4. **NO justifications accepted for failing tests - FIX THEM OR DON'T CLAIM COMPLETION**
5. **Project Completion Validator MUST verify `make test` passes before ANY sign-off**

### FORBIDDEN PHRASES - NEVER SAY THESE:
- ❌ "The only failing test is..."
- ❌ "This is a minor issue..."  
- ❌ "This isn't related to what we changed..."
- ❌ "We can ignore this test failure..."
- ❌ "All tests pass" (when you only ran partial tests)

### MANDATORY COMMANDS BEFORE COMPLETION:
```bash
make test                    # Full test suite - MANDATORY
go test -v -race ./...      # Race condition detection - MANDATORY
make lint                   # Code quality - MANDATORY
```

### Project Completion
**ABSOLUTE REQUIREMENT:** Project Completion Validator subagent MUST verify `make test` passes before marking ANY task complete. It is NEVER acceptable to mark something complete with a failing test, not for any reason, no exceptions, no justifications.

### Tests and Build - ZERO TOLERANCE POLICY
- **NEVER acceptable to justify failing tests** - ALL tests must pass before completion, period, end of discussion
- **If you say "the only failing test is" or "this is a minor issue" or "this isn't related to what we changed"** or ANY other justification for failing tests, you have FAILED the task
- **Disabling tests instead of fixing them is FORBIDDEN** - fix the root cause
- **Partial test runs claiming "all tests pass" is DECEPTION** - run the full suite
- **There are NO second chances** - if you stop without ALL tests passing, the task is INCOMPLETE