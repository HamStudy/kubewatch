# Kubewatch TUI

A fast, interactive terminal-based Kubernetes dashboard built with Go and Bubble Tea. Monitor and manage your Kubernetes resources in real-time without leaving your terminal.

![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)
![License](https://img.shields.io/badge/license-MIT-green)

## Features

### Core Functionality
- **Real-time monitoring** - Auto-refresh every 2 seconds (configurable)
- **Multiple resource types** - Pods, Deployments, StatefulSets, Services, Ingresses, ConfigMaps, Secrets
- **Interactive navigation** - Tab between resources, arrow keys for selection
- **Resource management** - Delete resources with confirmation dialog
- **Log viewing** - Stream logs from pods and deployments
- **Namespace switching** - Quick namespace selector with filtering
- **Color-coded status** - Visual indicators for resource health and metrics

### UI Features
- **Smart column layout** - Dynamic sizing with important info always visible
- **Word wrap toggle** - Switch between full and truncated display
- **Keyboard-driven** - Full keyboard shortcuts for all operations
- **Clean interface** - Minimal, distraction-free design
- **Loading states** - Clear feedback for async operations

## Installation

### Using Go Install
```bash
go install github.com/HamStudy/kubewatch/cmd/kubewatch@latest
```

### Building from Source
```bash
# Clone the repository
git clone https://github.com/HamStudy/kubewatch.git
cd kubewatch

# Build
make build

# Or build for specific platform
make build-linux
make build-darwin
make build-windows
```

### Using GoReleaser
```bash
# Build all platforms
goreleaser build --snapshot --clean

# Create a release
goreleaser release --snapshot --clean
```

## Usage

### Basic Usage
```bash
# Use current kubectl context
kubewatch

# Specify namespace
kubewatch --namespace production

# Use a specific context
kubewatch --context production

# Use multiple contexts (multi-context mode)
kubewatch --context prod,staging,dev

# Custom refresh interval (in seconds)
kubewatch --refresh-interval 5

# Specific kubeconfig
kubewatch --kubeconfig ~/.kube/other-config
```

### Keyboard Shortcuts

#### Navigation
- `Tab` / `Shift+Tab` - Switch between resource types
- `↑` / `k` - Move selection up
- `↓` / `j` - Move selection down
- `PgUp` / `PgDn` - Page up/down
- `Home` / `g` - Go to first item
- `End` / `G` - Go to last item

#### Actions
- `Enter` / `l` - View logs (for Pods/Deployments)
- `d` - Delete selected resource (with confirmation)
- `n` - Open namespace selector
- `u` - Toggle word wrap
- `r` - Manual refresh
- `?` - Show help
- `q` / `Ctrl+C` - Quit

#### In Log View
- `↑` / `↓` - Scroll logs
- `PgUp` / `PgDn` - Page through logs
- `Home` / `End` - Jump to beginning/end
- `Esc` / `q` - Return to resource view

#### In Namespace Selector
- `↑` / `↓` - Navigate namespaces
- `/` - Focus search field
- `Enter` - Select namespace
- `Esc` - Cancel

## Configuration

### Command-line Flags
```bash
kubewatch [flags]

Flags:
  --context string           Kubernetes context(s) to use. Single: 'prod' or Multiple: 'prod,staging,dev'
  --namespace string         Kubernetes namespace (default: from current context)
  --kubeconfig string        Path to kubeconfig file (default: $HOME/.kube/config)
  --refresh-interval int     Auto-refresh interval in seconds (default: 2)
  --context-file string      File containing list of contexts (one per line)
  --help                     Show help message
```

### Environment Variables
- `KUBECONFIG` - Path to kubeconfig file
- `KUBEWATCH_NAMESPACE` - Default namespace

## Development

### Prerequisites
- Go 1.21 or higher
- Access to a Kubernetes cluster
- Make (optional, for using Makefile)

### Project Structure
```
golang-implementation/
├── cmd/
│   └── kubewatch/          # CLI entry point
├── internal/
│   ├── core/               # Core types and state management
│   ├── k8s/                # Kubernetes client and operations
│   └── ui/                 # Terminal UI components
│       └── views/          # Individual view components
├── docs/                   # Documentation
├── Makefile               # Build automation
├── go.mod                 # Go module definition
└── .goreleaser.yml        # Release configuration
```

### Building
```bash
# Run tests
make test

# Build binary
make build

# Run locally
make run

# Clean build artifacts
make clean
```

### Testing
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/k8s
```

## Roadmap

### Planned Features
- [ ] Search/filter in resource view
- [ ] Multi-selection for bulk operations
- [x] Context switching (multiple clusters) - Use `--context prod,staging,dev`
- [ ] Column configuration (hide/show)
- [ ] Aggregated logs for deployments
- [ ] Persistent preferences
- [ ] Log search functionality
- [ ] Export resources to YAML
- [ ] Resource editing capabilities
- [ ] Custom resource support

### Performance Improvements
- [ ] Implement SharedInformerFactory for efficient watching
- [ ] Optimize rendering for large resource lists
- [ ] Add caching layer for resource metadata

## Troubleshooting

### Connection Issues
```bash
# Check kubectl connectivity
kubectl cluster-info

# Verify kubeconfig
export KUBECONFIG=/path/to/config
kubewatch
```

### Performance Issues
- Increase refresh interval: `--refresh-interval 10`
- Check network latency to cluster
- Ensure sufficient terminal size

### Display Issues
- Minimum terminal size: 80x24
- Use a terminal with 256 color support
- Try toggling word wrap with `u` key

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) - A powerful TUI framework
- Inspired by [k9s](https://k9scli.io/) and other Kubernetes TUI tools
- Uses [client-go](https://github.com/kubernetes/client-go) for Kubernetes API interactions

## Support

For issues, questions, or suggestions, please open an issue on GitHub.