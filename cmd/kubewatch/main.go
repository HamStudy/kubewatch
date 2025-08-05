package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/kubewatch-tui/internal/core"
	"github.com/user/kubewatch-tui/internal/k8s"
	"github.com/user/kubewatch-tui/internal/ui"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// CLIFlags holds all command-line flags
type CLIFlags struct {
	// Kubernetes connection flags
	kubeconfig    string
	context       string
	namespace     string
	allNamespaces bool

	// Authentication flags
	user                 string
	cluster              string
	authInfoName         string
	clientCertificate    string
	clientKey            string
	certificateAuthority string
	insecureSkipVerify   bool
	token                string
	tokenFile            string
	asUser               string
	asGroup              []string
	asUID                string

	// Request flags
	timeout        string
	requestTimeout string

	// UI flags
	refreshInterval   int
	logTailLines      int
	maxResourcesShown int
	colorScheme       string
	resourceType      string // Initial resource type to display

	// Other flags
	version  bool
	help     bool
	verbose  bool
	logLevel string
	cacheDir string
}

func parseFlags() *CLIFlags {
	flags := &CLIFlags{}

	// Define flags similar to kubectl
	flag.StringVar(&flags.kubeconfig, "kubeconfig", "", "Path to the kubeconfig file to use for CLI requests (can also use KUBECONFIG env var)")
	flag.StringVar(&flags.context, "context", "", "The name of the kubeconfig context to use")
	flag.StringVar(&flags.namespace, "namespace", "", "If present, the namespace scope for this CLI request")
	flag.StringVar(&flags.namespace, "n", "", "Shorthand for --namespace")
	flag.BoolVar(&flags.allNamespaces, "all-namespaces", false, "If present, list the requested object(s) across all namespaces")
	flag.BoolVar(&flags.allNamespaces, "A", false, "Shorthand for --all-namespaces")

	// Authentication flags
	flag.StringVar(&flags.user, "user", "", "The name of the kubeconfig user to use")
	flag.StringVar(&flags.cluster, "cluster", "", "The name of the kubeconfig cluster to use")
	flag.StringVar(&flags.authInfoName, "auth-info-name", "", "The name of the kubeconfig auth info to use")
	flag.StringVar(&flags.clientCertificate, "client-certificate", "", "Path to a client certificate file for TLS")
	flag.StringVar(&flags.clientKey, "client-key", "", "Path to a client key file for TLS")
	flag.StringVar(&flags.certificateAuthority, "certificate-authority", "", "Path to a cert file for the certificate authority")
	flag.BoolVar(&flags.insecureSkipVerify, "insecure-skip-tls-verify", false, "If true, the server's certificate will not be checked for validity")
	flag.StringVar(&flags.token, "token", "", "Bearer token for authentication to the API server")
	flag.StringVar(&flags.tokenFile, "token-file", "", "Path to a file containing a bearer token for authentication")
	flag.StringVar(&flags.asUser, "as", "", "Username to impersonate for the operation")
	flag.StringVar(&flags.asUID, "as-uid", "", "UID to impersonate for the operation")

	// Request flags
	flag.StringVar(&flags.timeout, "timeout", "0s", "The length of time to wait before giving up on a single server request")
	flag.StringVar(&flags.requestTimeout, "request-timeout", "0s", "The length of time to wait before giving up on a single server request")

	// UI-specific flags
	flag.IntVar(&flags.refreshInterval, "refresh-interval", 2, "Refresh interval in seconds for updating resources")
	flag.IntVar(&flags.logTailLines, "log-tail-lines", 100, "Number of log lines to tail when viewing logs")
	flag.IntVar(&flags.maxResourcesShown, "max-resources", 500, "Maximum number of resources to display")
	flag.StringVar(&flags.colorScheme, "color-scheme", "default", "Color scheme to use (default, dark, light)")

	// Other flags
	flag.BoolVar(&flags.version, "version", false, "Print version information and quit")
	flag.BoolVar(&flags.version, "v", false, "Shorthand for --version")
	flag.BoolVar(&flags.help, "help", false, "Show help message")
	flag.BoolVar(&flags.help, "h", false, "Shorthand for --help")
	flag.BoolVar(&flags.verbose, "verbose", false, "Enable verbose output")
	flag.StringVar(&flags.logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.StringVar(&flags.cacheDir, "cache-dir", "", "Default cache directory")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Kubewatch TUI - Terminal-based Kubernetes Dashboard\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  kubewatch [flags] [resource-type]\n\n")
		fmt.Fprintf(os.Stderr, "Resource Types:\n")
		fmt.Fprintf(os.Stderr, "  pods, pod, po          - Show pods (default)\n")
		fmt.Fprintf(os.Stderr, "  deployments, deploy    - Show deployments\n")
		fmt.Fprintf(os.Stderr, "  statefulsets, sts      - Show statefulsets\n")
		fmt.Fprintf(os.Stderr, "  services, svc          - Show services\n")
		fmt.Fprintf(os.Stderr, "  ingresses, ing         - Show ingresses\n")
		fmt.Fprintf(os.Stderr, "  configmaps, cm         - Show configmaps\n")
		fmt.Fprintf(os.Stderr, "  secrets                - Show secrets\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  # Use kubewatch with default kubeconfig\n")
		fmt.Fprintf(os.Stderr, "  kubewatch\n\n")
		fmt.Fprintf(os.Stderr, "  # Start with deployments view\n")
		fmt.Fprintf(os.Stderr, "  kubewatch deployments\n\n")
		fmt.Fprintf(os.Stderr, "  # Use specific context and namespace\n")
		fmt.Fprintf(os.Stderr, "  kubewatch --context=production --namespace=web\n\n")
		fmt.Fprintf(os.Stderr, "  # Use custom kubeconfig file\n")
		fmt.Fprintf(os.Stderr, "  kubewatch --kubeconfig=/path/to/config\n\n")
		fmt.Fprintf(os.Stderr, "  # Watch deployments in prod namespace\n")
		fmt.Fprintf(os.Stderr, "  kubewatch -n prod deployments\n\n")
		fmt.Fprintf(os.Stderr, "  # Watch all namespaces\n")
		fmt.Fprintf(os.Stderr, "  kubewatch --all-namespaces\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nKeyboard Shortcuts:\n")
		fmt.Fprintf(os.Stderr, "  Tab        - Switch between resource types\n")
		fmt.Fprintf(os.Stderr, "  j/k        - Navigate up/down\n")
		fmt.Fprintf(os.Stderr, "  g/G        - Go to top/bottom\n")
		fmt.Fprintf(os.Stderr, "  d          - Delete selected resource\n")
		fmt.Fprintf(os.Stderr, "  l          - View logs (pods only)\n")
		fmt.Fprintf(os.Stderr, "  n          - Change namespace\n")
		fmt.Fprintf(os.Stderr, "  /          - Search/filter resources\n")
		fmt.Fprintf(os.Stderr, "  ?          - Show help\n")
		fmt.Fprintf(os.Stderr, "  q/Ctrl+C   - Quit\n")
	}

	flag.Parse()

	// Handle multi-value flags
	flag.Func("as-group", "Group to impersonate for the operation (can be repeated)", func(s string) error {
		flags.asGroup = append(flags.asGroup, s)
		return nil
	})

	// Check for positional argument (resource type)
	args := flag.Args()
	if len(args) > 0 {
		// First positional argument is the resource type
		flags.resourceType = args[0]
	}

	return flags
}

func main() {
	flags := parseFlags()

	// Handle version flag
	if flags.version {
		fmt.Printf("kubewatch version %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
		os.Exit(0)
	}

	// Handle help flag
	if flags.help {
		flag.Usage()
		os.Exit(0)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Initialize configuration with CLI flags
	config, err := loadConfigWithFlags(flags)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Kubernetes client with additional options
	k8sClient, err := k8s.NewClientWithOptions(config.KubeConfig, &k8s.ClientOptions{
		Context:              flags.context,
		Namespace:            config.CurrentNamespace,
		User:                 flags.user,
		Cluster:              flags.cluster,
		ClientCertificate:    flags.clientCertificate,
		ClientKey:            flags.clientKey,
		CertificateAuthority: flags.certificateAuthority,
		InsecureSkipVerify:   flags.insecureSkipVerify,
		Token:                flags.token,
		TokenFile:            flags.tokenFile,
		Impersonate:          flags.asUser,
		ImpersonateGroups:    flags.asGroup,
		ImpersonateUID:       flags.asUID,
		Timeout:              flags.timeout,
		CacheDir:             flags.cacheDir,
	})
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	// Initialize application state
	state := core.NewState(config)

	// Create the main application
	app := ui.NewApp(ctx, k8sClient, state, config)

	// Create Bubble Tea program
	p := tea.NewProgram(app, tea.WithAltScreen())

	// Run the application
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running application: %v", err)
	}
}

// loadConfigWithFlags loads configuration with CLI flag overrides
func loadConfigWithFlags(flags *CLIFlags) (*core.Config, error) {
	// Load base configuration
	config, err := core.LoadConfig()
	if err != nil {
		return nil, err
	}

	// Override with CLI flags
	if flags.kubeconfig != "" {
		config.KubeConfig = flags.kubeconfig
	}

	if flags.namespace != "" {
		config.CurrentNamespace = flags.namespace
	} else if flags.allNamespaces {
		config.CurrentNamespace = "" // Empty namespace means all namespaces
	}

	if flags.context != "" {
		config.CurrentContext = flags.context
	}

	if flags.refreshInterval > 0 {
		config.RefreshInterval = flags.refreshInterval
	}

	if flags.logTailLines > 0 {
		config.LogTailLines = flags.logTailLines
	}

	if flags.maxResourcesShown > 0 {
		config.MaxResourcesShown = flags.maxResourcesShown
	}

	if flags.colorScheme != "" {
		config.ColorScheme = flags.colorScheme
	}

	// Set initial resource type if specified
	if flags.resourceType != "" {
		// Parse resource type aliases
		switch strings.ToLower(flags.resourceType) {
		case "pods", "pod", "po":
			config.InitialResourceType = "pod"
		case "deployments", "deployment", "deploy":
			config.InitialResourceType = "deployment"
		case "statefulsets", "statefulset", "sts":
			config.InitialResourceType = "statefulset"
		case "services", "service", "svc":
			config.InitialResourceType = "service"
		case "ingresses", "ingress", "ing":
			config.InitialResourceType = "ingress"
		case "configmaps", "configmap", "cm":
			config.InitialResourceType = "configmap"
		case "secrets", "secret":
			config.InitialResourceType = "secret"
		default:
			// Default to the provided value
			config.InitialResourceType = flags.resourceType
		}
	}

	// Handle cache directory
	if flags.cacheDir != "" {
		// Set cache directory if needed
	} else if flags.cacheDir == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			flags.cacheDir = filepath.Join(home, ".kube", "cache")
		}
	}

	return config, nil
}
