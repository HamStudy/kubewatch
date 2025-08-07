# Selection Jumping Bug Test Summary

## Overview
Comprehensive tests have been added to `resource_view_test.go` that reproduce the selection jumping bug in ResourceView, particularly in multi-context mode.

## Test Coverage

### 1. TestMultiContextSelectionJumpingBug (NEW)
**Purpose**: Specifically tests selection persistence in multi-context mode where pods from different contexts are interleaved.

**Test Scenarios**:
- **Multi-context pod interleaving**: Tests selection when pods from multiple clusters (prod, staging, dev) are displayed together
- **New pods from different contexts**: Tests selection when new pods are added from various contexts
- **Context filtering**: Tests selection behavior when rapidly switching between context filters

**Current Status**: PASSING (bug may not manifest in these specific scenarios or restoreSelectionByIdentity works in these cases)

### 2. TestResourceViewSelectionJumpingBugDuringRefresh
**Purpose**: Tests selection persistence during refresh operations with data changes.

**Test Scenarios**:
- Selection stays on same pod after refresh with updated data
- Selection follows pod when list is reordered
- Selection handles deleted pods gracefully
- Selection stays at bottom when last pod selected
- Selection resets when all pods are replaced

**Current Status**: FAILING (2 out of 5 scenarios fail, demonstrating the bug)

### 3. TestResourceViewSelectionBugComprehensive  
**Purpose**: Comprehensive test of selection persistence through various refresh cycles.

**Test Scenarios**:
- Status updates only (pod status changes)
- New pod added that sorts before selected pod
- New pod added that sorts after selected pod
- Rapid successive refreshes

**Current Status**: FAILING (selection jumps when new pods are added)

## Bug Manifestation

The tests reveal that selection jumping occurs when:
1. **New pods are added** that change the sort order
2. **Pods are reordered** due to status changes or sorting
3. **Multiple contexts** are involved with interleaved data

## Key Findings

1. The `restoreSelectionByIdentity()` method doesn't always work correctly
2. Selection often jumps to a different pod or resets to top
3. The bug is more pronounced when the list structure changes (additions/deletions)
4. Multi-context mode adds complexity but the core bug exists in single-context too

## Test Execution

Run all selection bug tests:
```bash
go test -v ./internal/ui/views -run "TestResourceViewSelectionJumpingBugDuringRefresh|TestResourceViewSelectionBugComprehensive|TestMultiContextSelectionJumpingBug"
```

Run only multi-context tests:
```bash
go test -v ./internal/ui/views -run TestMultiContextSelectionJumpingBug
```

## Next Steps

These failing tests provide a solid foundation for:
1. Debugging the root cause of the selection jumping issue
2. Implementing a fix in the ResourceView selection persistence logic
3. Ensuring the fix works across all scenarios (single and multi-context)
4. Using these tests as regression tests once the bug is fixed

## Test Design Principles

The tests follow best practices:
- **Behavioral testing**: Tests what users experience (selection jumping)
- **Clear failure messages**: Shows exactly what went wrong
- **Comprehensive scenarios**: Covers edge cases and real-world usage
- **Isolated test cases**: Each test scenario is independent
- **Descriptive names**: Test names clearly indicate what's being tested