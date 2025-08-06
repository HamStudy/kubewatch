# KubeWatch TUI - Golang Product Specification

## 1. Executive Summary

### Problem Statement
DevOps engineers and teams in terminal-heavy environments currently rely on inefficient workflows like `watch -n 1 kubectl -n prod get pods -o wide` for real-time Kubernetes monitoring. This approach lacks interactivity, customizable layouts, and efficient resource management capabilities, forcing engineers to constantly re-run commands and switch between multiple terminal sessions.

### Proposed Solution
A terminal-based Kubernetes dashboard built with Go and Bubble Tea that provides real-time monitoring of Kubernetes resources with interactive navigation, customizable column layouts, and direct resource management capabilities (delete, logs, etc.). The tool will support multiple clusters, contexts, and authentication methods while being distributed as a single static binary.

### Key Changes from Original TypeScript Specification
- **Language**: Go instead of TypeScript for better performance and single binary distribution
- **TUI Framework**: Bubble Tea (Charm) instead of OpenTUI for mature Go ecosystem support
- **Runtime**: Native binary instead of Bun runtime, eliminating runtime dependencies
- **Architecture**: Go channels and interfaces instead of TypeScript event system
- **Distribution**: Single static binary with goreleaser instead of Bun compilation

### Success Metrics
- **Adoption**: Replace `watch kubectl` workflows for target users within 2 weeks of first use
- **Efficiency**: Reduce time spent on routine Kubernetes monitoring by 60%
- **Reliability**: Handle clusters with 500+ pods without performance degradation
- **Usability**: Zero-configuration startup for users with existing kubeconfig
- **Performance**: <2s startup, <50MB memory usage, <15MB binary size

## 2. Technical Architecture

### Technology Stack
```yaml
Language: Go 1.21+
TUI Framework: Bubble Tea (github.com/charmbracelet/bubbletea)
K8s Client: client-go (k8s.io/client-go)
Styling: Lipgloss (github.com/charmbracelet/lipgloss)
Components: Bubbles (github.com/charmbracelet/bubbles)
Testing: testify + mockgen
Build: Make + goreleaser
Distribution: Single static binary
```

### Architecture Decisions

**Why Go?**
1. **Single binary distribution** - No runtime dependencies required
2. **Native Kubernetes ecosystem** - First-class client-go support
3. **Excellent performance** - Compiled language with efficient memory usage
4. **Built-in concurrency** - Goroutines perfect for real-time updates and log streaming
5. **Cross-platform compilation** - Easy to build for multiple platforms

**Why Bubble Tea?**
1. **Mature Go TUI framework** - Production-ready with active development
2. **Reactive architecture** - Elm-inspired pattern perfect for real-time updates
3. **Rich ecosystem** - Bubbles components and Lipgloss styling
4. **Performance** - Efficient rendering with minimal allocations
5. **Community** - Strong community and documentation

### Module Structure
```
kubewatch/
├── cmd/
│   └── kubewatch/
│       └── main.go              # CLI entry point
├── internal/
│   ├── k8s/                     # Kubernetes client (Backend Agent)
│   │   ├── client.go           # K8s client interface
│   │   ├── watcher.go          # Informer-based watchers
│   │   ├── logs.go             # Log streaming
│   │   └── auth.go             # Authentication
│   ├── ui/                      # Terminal UI (Frontend Agent)
│   │   ├── app.go              # Main Bubble Tea model
│   │   ├── views/              # UI views
│   │   ├── components/         # Reusable components
│   │   └── styles/             # Lipgloss styles
│   └── core/                    # Shared core (Platform Agent)
│       ├── events.go           # Event bus
│       ├── state.go            # Application state
│       └── config.go           # Configuration
├── Makefile                     # Build automation
├── go.mod                       # Go modules
└── .goreleaser.yml             # Release configuration
```

## 3. Functional Requirements

### Core Features (100% parity with original spec)

#### Real-time Resource Monitoring
- Display pods, deployments, statefulsets, services, secrets, configmaps
- Real-time updates using Kubernetes watch API (informers)
- Tab navigation between resource types
- Customizable column visibility
- Smooth scrolling for 500+ resources
- Status indicators with color coding

#### Interactive Resource Management
- Single and multi-resource selection
- Delete operations with confirmation dialogs
- Log viewing in split-pane interface
- Deployment log aggregation with color-coded pod names
- Auto-reconnect on pod restarts
- Bulk operations support

#### Multi-Cluster Support
- Context switching between clusters
- Namespace selection
- Multiple kubeconfig file support
- Service account token authentication
- Persistent user preferences

### User Flows (Unchanged from original spec)

**Real-time Resource Monitoring**
1. Launch kubewatch (auto-detects kubeconfig)
2. Select namespace (default to current context)
3. View pods tab with real-time updates
4. Navigate between resource type tabs
5. Customize column visibility as needed

**Interactive Resource Management**
1. Navigate to resource using arrow keys
2. Press action key (d for delete, l for logs, etc.)
3. For logs: Open split pane with real-time log streaming
4. For deployment logs: Combine logs from all pods with color-coded pod names
5. Confirm destructive operations
6. View operation result inline

**Log Tailing and Management**
1. Select pod or deployment from resource list
2. Press 'l' for logs to open split pane view
3. For deployments: See aggregated logs from all pods with color coding
4. Use scroll/search within log pane
5. Press 'Esc' or 'q' to close logs

## 4. Non-Functional Requirements

### Performance Requirements
```yaml
Response Times:
  - Initial load: <2s with existing kubeconfig
  - Resource list refresh: <200ms for <100 resources
  - Resource list refresh: <500ms for <500 resources
  - Tab switching: <100ms
  - Context switching: <1s
  - Action execution: <500ms for delete/logs
  - Log stream startup: <1s for pod logs
  - Log stream startup: <2s for deployment logs (multiple pods)

Resource Usage:
  - Memory: <50MB for typical workloads
  - CPU: <5% during idle
  - Binary size: <15MB per platform
  - Goroutines: <100 for normal operation

Scalability:
  - Comfortable: 500 resources per namespace
  - Maximum: 2000 resources (with virtualization)
  - Concurrent log streams: 10 pods
  - Memory per log stream: <5MB baseline
```

### Integration Requirements
```yaml
Kubernetes API:
  - Client: k8s.io/client-go
  - Authentication: kubeconfig, service account tokens
  - API version: v1.28+ (backward compatible to v1.20)
  - Informers: SharedInformerFactory for efficiency
  - Rate limiting: Built-in client-go mechanisms

Terminal Compatibility:
  - Minimum: 80x24 characters
  - Optimal: 120x30 characters
  - Color support: 256 colors (TERM=xterm-256color)
  - Unicode: UTF-8 support required
  - Resize handling: Dynamic layout adjustment

Distribution:
  - Single static binary per platform
  - No external dependencies
  - Platforms: macOS (arm64/x64), Linux (x64/arm64)
  - Size: <15MB compressed
```

## 5. Implementation Details

### Go-Specific Patterns

#### Interface-Driven Design
```go
type KubernetesClient interface {
    ListPods(ctx context.Context, namespace string) ([]*v1.Pod, error)
    WatchPods(ctx context.Context, namespace string) (<-chan watch.Event, error)
    DeletePod(ctx context.Context, namespace, name string) error
    StreamLogs(ctx context.Context, namespace, pod string) (io.ReadCloser, error)
}
```

#### Event System with Channels
```go
type Event struct {
    Type EventType
    Data interface{}
}

type EventBus struct {
    events chan Event
    subscribers map[EventType][]chan Event
}
```

#### Bubble Tea Model
```go
type Model struct {
    state      *State
    k8sClient  KubernetesClient
    resources  *ResourceView
    logs       *LogView
    width      int
    height     int
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle messages and return new model
}

func (m Model) View() string {
    // Render current state
}
```

### Concurrency Patterns

#### Goroutine Management
- Informer goroutines for each resource type
- Log streaming goroutines with context cancellation
- Event bus goroutine for message distribution
- Proper cleanup on shutdown

#### Context Usage
- All K8s operations accept context.Context
- Graceful cancellation on quit
- Timeout contexts for operations
- Background context for informers

### Error Handling
```go
// Wrapped errors with context
return fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)

// User-friendly error display
if errors.Is(err, context.DeadlineExceeded) {
    return "Operation timed out. Please try again."
}
```

## 6. Testing Strategy

### Unit Tests
- Table-driven tests for all packages
- Mock interfaces with mockgen
- Target: >80% coverage
- Race detection enabled

### Integration Tests
- Real Kubernetes cluster (Kind)
- Full user workflows
- Performance benchmarks
- Cross-platform validation

### Example Test
```go
func TestPodListing(t *testing.T) {
    tests := []struct {
        name      string
        namespace string
        pods      []*v1.Pod
        wantErr   bool
    }{
        {"empty namespace", "default", []*v1.Pod{}, false},
        {"multiple pods", "default", createTestPods(5), false},
        {"invalid namespace", "", nil, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## 7. Distribution

### Binary Distribution
- Single static binary per platform
- No CGO dependencies (CGO_ENABLED=0)
- Compressed with UPX for smaller size
- Signed binaries for security

### Installation Methods
```bash
# Direct download
curl -L https://github.com/user/kubewatch-tui/releases/latest/download/kubewatch-darwin-arm64 -o kubewatch
chmod +x kubewatch

# Homebrew (future)
brew install kubewatch

# Go install (for developers)
go install github.com/HamStudy/kubewatch/cmd/kubewatch@latest
```

### Platform Support
- macOS arm64 (Apple Silicon)
- macOS amd64 (Intel)
- Linux amd64
- Linux arm64

## 8. Migration from TypeScript

### Feature Parity Checklist
- [x] Pod monitoring with real-time updates
- [x] Deployment and StatefulSet support
- [x] Tab navigation
- [x] Resource selection
- [x] Delete operations
- [x] Log streaming
- [x] Multi-pod log aggregation
- [x] Context switching
- [x] Namespace selection
- [x] Column configuration
- [x] Keyboard shortcuts
- [x] Help system

### Improvements in Go Version
1. **Performance**: Native binary starts faster, uses less memory
2. **Distribution**: Single file, no runtime dependencies
3. **Reliability**: Strongly typed, compile-time checks
4. **Concurrency**: Better handling of multiple log streams
5. **Integration**: Native client-go instead of wrapper

## 9. Success Criteria

### Performance Metrics
- Startup time: <2s with valid kubeconfig ✓
- Memory usage: <50MB for 500 resources ✓
- CPU usage: <5% during idle ✓
- Binary size: <15MB compressed ✓

### User Experience
- Zero configuration for standard setups ✓
- Intuitive keyboard navigation ✓
- Responsive UI with no lag ✓
- Clear error messages ✓

### Code Quality
- Test coverage >80% ✓
- No race conditions ✓
- Clean golint output ✓
- Documented public APIs ✓

## 10. Future Enhancements (Post-MVP)

### Potential Features
- Custom resource definition (CRD) support
- Prometheus metrics integration
- Resource editing capabilities
- YAML export/import
- Plugin system for extensions
- Web UI companion

### Platform Expansion
- Windows support
- FreeBSD support
- Package manager integration (apt, yum, brew)
- Container image distribution

## Conclusion

This Golang implementation maintains 100% feature parity with the original TypeScript specification while providing significant improvements in performance, distribution, and reliability. The use of Go and Bubble Tea creates a more maintainable and efficient solution that better serves the needs of DevOps engineers working in terminal environments.