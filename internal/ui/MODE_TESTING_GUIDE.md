# Mode Testing Guide for KubeWatch TUI

## Overview
This guide explains how the mode system works in KubeWatch TUI and provides best practices for testing mode-related functionality.

## Mode System Architecture

### Key Components

1. **ScreenMode Interface** (`modes.go`)
   - Defines the contract for all screen modes
   - Methods: `GetType()`, `GetKeyBindings()`, `HandleKey()`, `GetHelpSections()`, `GetTitle()`

2. **Mode Types** (7 modes total)
   - `ModeList` - Main resource list view
   - `ModeLog` - Log streaming view
   - `ModeDescribe` - Resource description view
   - `ModeHelp` - Help screen
   - `ModeContextSelector` - Context selection dialog
   - `ModeNamespaceSelector` - Namespace selection dialog
   - `ModeConfirmDialog` - Confirmation dialogs (e.g., delete)

3. **Key Handling Flow**
   ```
   User Input → App.Update() → Mode.HandleKey() → View.Update()
                                    ↓
                              (handled/not handled)
                                    ↓
                            If not handled, delegate to view
   ```

## Testing Challenges and Solutions

### Challenge 1: Testing Keys That Require Selected Resources

**Problem**: Some key handlers check `app.resourceView.GetSelectedResourceName()` which depends on internal view state that cannot be mocked through public APIs.

**Solution**: 
- Skip these tests at the unit level with clear documentation
- Test these behaviors in integration tests where full app state can be set up
- Consider adding a testing interface to ResourceView if this becomes a common need

**Example**:
```go
// TestListModeKeyHandlingWithSelectedResource
t.Skip("Skipping test that requires mocking internal resource view state")
```

### Challenge 2: Testing Search Mode in LogView

**Problem**: The log view's search mode is internal state accessed via `IsSearchMode()` but cannot be set externally for testing.

**Solution**:
- Comment out tests that require manipulating internal search state
- Test search mode behavior through integration tests
- Consider exposing a test-only method if needed frequently

### Challenge 3: Nil Pointer Issues with Views

**Problem**: Mode handlers may call methods on views (like `namespaceView`) that are nil in test setup.

**Solution**:
- Always initialize required views before testing mode handlers
- Use helper functions to set up views with test data

**Example**:
```go
// Initialize namespace view to prevent nil pointer
namespaces := []v1.Namespace{
    {ObjectMeta: metav1.ObjectMeta{Name: "default"}},
}
app.namespaceView = views.NewNamespaceView(namespaces, app.state.CurrentNamespace)
```

## Best Practices for Mode Testing

### 1. Test Mode Transitions
Focus on testing that modes transition correctly rather than testing internal state changes:

```go
func TestModeTransitions(t *testing.T) {
    app := createTestApp(t)
    app.setMode(ModeList)
    app.setMode(ModeHelp)
    
    if app.currentMode != ModeHelp {
        t.Error("Mode should transition to Help")
    }
}
```

### 2. Test Key Binding Coverage
Ensure all defined key bindings are tested:

```go
func TestListModeCompleteKeyHandling(t *testing.T) {
    tests := []struct {
        name          string
        keyType       tea.KeyType
        keyRunes      []rune
        expectHandled bool
        expectMode    ScreenModeType
    }{
        // Test each key binding...
    }
}
```

### 3. Use Table-Driven Tests
Table-driven tests make it easy to add new test cases and see coverage:

```go
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test logic
    })
}
```

### 4. Initialize Dependencies
Always initialize view dependencies to avoid nil pointers:

```go
app.namespaceView = views.NewNamespaceView(namespaces, currentNamespace)
app.contextView = views.NewContextView(contexts, selected)
app.confirmView = views.NewConfirmView(title, message)
```

### 5. Test Helper Functions
Use the provided test helpers in `test_helpers.go` and `mode_test_helpers.go`:

```go
// Create a properly initialized test app
app := createTestApp(t)

// Set up mode-specific dependencies
setup := NewModeTestSetup(t, ModeNamespaceSelector)
```

## Testing Anti-Patterns to Avoid

### ❌ Don't Test Private State
Avoid testing or depending on private fields:
```go
// BAD: Trying to access private fields
app.resourceView.selectedRow = 5  // Won't compile
```

### ❌ Don't Test Implementation Details
Focus on behavior, not how it's implemented:
```go
// BAD: Testing that a specific internal method was called
// GOOD: Testing that the visible behavior changed
```

### ❌ Don't Create Brittle Tests
Avoid tests that break with minor refactoring:
```go
// BAD: Testing exact string output that might change
// GOOD: Testing that key elements are present
```

## Integration vs Unit Testing

### Unit Tests (modes_test.go)
- Test individual mode key handlers
- Test mode transitions
- Test help text generation
- Skip tests requiring complex state setup

### Integration Tests (integration_test.go, workflow_integration_test.go)
- Test complete user workflows
- Test mode interactions with real views
- Test resource selection and actions
- Test multi-context scenarios

## Common Test Patterns

### Pattern 1: Testing Key Delegation
```go
// Test that navigation keys are delegated to views
{"up arrow", tea.KeyUp, nil, false, ModeList, "Should delegate to view"},
```

### Pattern 2: Testing Mode Changes
```go
// Test that certain keys trigger mode changes
{"help", tea.KeyRunes, []rune("?"), true, ModeHelp, "Should switch to help"},
```

### Pattern 3: Testing Quit Behavior
```go
// Test that quit keys are handled consistently
{"quit q", tea.KeyRunes, []rune("q"), true, ModeList, "Should handle quit"},
```

## Future Improvements

1. **Testing Interface for Views**: Consider adding a testing interface to views that allows setting internal state for testing purposes.

2. **Mock Mode System**: Create a mock mode system for testing mode transitions without full app setup.

3. **Behavioral Testing Framework**: Build a higher-level testing framework that focuses on user behaviors rather than implementation details.

4. **Test Coverage Metrics**: Add coverage tracking specifically for key bindings to ensure all defined keys are tested.

## Summary

The mode system is central to KubeWatch TUI's user interaction model. Testing it effectively requires:
- Understanding the separation between modes and views
- Properly initializing dependencies
- Focusing on behavioral testing over implementation details
- Using integration tests for complex scenarios
- Following the established patterns in the codebase

When adding new modes or modifying existing ones, ensure:
1. All key bindings are documented and tested
2. Mode transitions work correctly
3. Views are properly initialized before testing
4. Tests are maintainable and focused on behavior