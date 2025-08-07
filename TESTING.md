# Testing Guide for KubeWatch TUI

This document describes the testing approach and setup for the KubeWatch TUI application.

## Overview

The KubeWatch TUI uses a comprehensive testing strategy to ensure reliability and prevent regressions:

- **Unit Tests**: Test individual components and functions
- **Integration Tests**: Test component interactions and workflows
- **Mode System Tests**: Test the screen mode system and key handling
- **UI Tests**: Test view rendering and state management

## Running Tests

### All Tests
```bash
make test
```

### UI Tests Only
```bash
make test-ui
```

### With Coverage
```bash
make coverage
```

### Individual Test Files
```bash
go test ./internal/ui/app_test.go -v
go test ./internal/ui/modes_test.go -v
```

## Test Structure

```
internal/ui/
├── app_test.go          # Main application tests
├── modes_test.go        # Mode system and key handling tests
├── test_helpers.go      # Test utilities and helpers
└── views/
    └── (future view-specific tests)
```

## Test Categories

### 1. Application Tests (`app_test.go`)

- **Initialization**: Tests app startup and initial state
- **Key Handling**: Tests key processing and command generation
- **View Rendering**: Tests that views render without errors
- **Window Sizing**: Tests responsive layout handling
- **Mode System**: Tests mode initialization and switching
- **Resource Navigation**: Tests resource type cycling
- **Error Handling**: Tests graceful error handling
- **State Consistency**: Tests internal state management

### 2. Mode System Tests (`modes_test.go`)

- **Key Binding Tests**: Tests that each mode handles its defined keys
- **Mode Transitions**: Tests switching between different modes
- **Key Binding Definitions**: Tests that all key bindings are properly defined
- **Help System**: Tests help text generation and display

### 3. Test Helpers (`test_helpers.go`)

- **Mock Setup**: Creates test applications with minimal dependencies
- **Test Utilities**: Helper functions for common test operations
- **Assertions**: Custom assertion functions for TUI-specific testing

## Key Testing Patterns

### 1. Mode Key Handling Tests

```go
func TestListModeKeyHandling(t *testing.T) {
    mode := NewListMode()
    app := createTestApp(t)
    
    tests := []struct {
        name          string
        key           string
        expectHandled bool
        expectMode    ScreenModeType
    }{
        {"help key", "?", true, ModeHelp},
        {"unknown key", "x", false, ModeList},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test key handling logic
        })
    }
}
```

### 2. Application Integration Tests

```go
func TestAppKeyHandling(t *testing.T) {
    app := createTestApp(t)
    
    // Create key message
    keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
    
    // Update app with key message
    model, cmd := app.Update(keyMsg)
    app = model.(*App)
    
    // Verify expected behavior
    if app.currentMode != ModeHelp {
        t.Errorf("Expected help mode")
    }
}
```

### 3. View Rendering Tests

```go
func TestAppViewRendering(t *testing.T) {
    app := createTestApp(t)
    
    // Test that view renders without panicking
    view := app.View()
    if len(view) == 0 {
        t.Error("View should not be empty")
    }
}
```

## Testing Best Practices

### 1. Test Isolation
- Each test creates its own app instance
- Tests don't depend on external state
- Mock dependencies are used where appropriate

### 2. Comprehensive Coverage
- Test both success and failure paths
- Test edge cases and error conditions
- Test all key bindings and mode transitions

### 3. Maintainable Tests
- Use descriptive test names
- Group related tests in subtests
- Use helper functions to reduce duplication

### 4. Fast Execution
- Tests run quickly without external dependencies
- Use minimal setup for each test
- Avoid unnecessary complexity

## Regression Prevention

### 1. Automated Testing
- All tests run on every change
- CI/CD integration ensures tests pass before merging
- Coverage reports track test completeness

### 2. Key Binding Validation
- Tests verify all key bindings are defined
- Tests ensure keys are handled consistently
- Tests check help text is complete

### 3. Mode System Validation
- Tests verify all modes are properly initialized
- Tests ensure mode transitions work correctly
- Tests validate mode-specific behavior

## Future Enhancements

### 1. Visual Testing
- Consider adding screenshot-based testing
- Test terminal output formatting
- Validate color and styling

### 2. Performance Testing
- Add benchmarks for view rendering
- Test with large datasets
- Monitor memory usage

### 3. End-to-End Testing
- Test complete user workflows
- Test with real Kubernetes clusters
- Validate error handling with network issues

## Running Tests in CI/CD

The tests are designed to run in automated environments:

```yaml
# Example GitHub Actions workflow
- name: Run tests
  run: make test

- name: Run UI tests
  run: make test-ui

- name: Generate coverage
  run: make coverage
```

## Debugging Tests

### 1. Verbose Output
```bash
go test ./internal/ui/... -v
```

### 2. Run Specific Tests
```bash
go test ./internal/ui/... -run TestAppKeyHandling
```

### 3. Debug Mode
```bash
go test ./internal/ui/... -v -count=1
```

## Contributing

When adding new features:

1. **Write tests first** - Use TDD approach when possible
2. **Test key bindings** - Ensure all new keys are tested
3. **Test mode behavior** - Verify mode-specific functionality
4. **Update documentation** - Keep this guide current
5. **Run all tests** - Ensure no regressions

## Test Coverage Goals

- **Overall**: >80% code coverage
- **UI Package**: >90% code coverage
- **Key Handling**: 100% coverage of all defined key bindings
- **Mode System**: 100% coverage of all mode transitions