package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
	"github.com/aspectrr/fluid.sh/fluid/internal/libvirt"
	"github.com/aspectrr/fluid.sh/fluid/internal/sshca"
	"github.com/aspectrr/fluid.sh/fluid/internal/sshkeys"
	"github.com/aspectrr/fluid.sh/fluid/internal/store"
	"github.com/aspectrr/fluid.sh/fluid/internal/store/sqlite"
	"github.com/aspectrr/fluid.sh/fluid/internal/telemetry"
	"github.com/aspectrr/fluid.sh/fluid/internal/tui"
	"github.com/aspectrr/fluid.sh/fluid/internal/vm"
)

var (
	cfgFile          string
	outputJSON       bool
	cfg              *config.Config
	dataStore        store.Store
	vmService        *vm.Service
	libvirtMgr       libvirt.Manager
	telemetryService telemetry.Service
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		outputError(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "fluid",
	Short: "Fluid - Embedded CLI for AI agent sandboxes",
	Long: `Fluid is a CLI tool that lets AI agents create and manage VM sandboxes.
Local SQLite for state, direct libvirt access via local socket or SSH.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip init for these commands (they handle their own init)
		if cmd.Name() == "init" || cmd.Name() == "version" || cmd.Name() == "help" || cmd.Name() == "tui" || cmd.Name() == "fluid" {
			return nil
		}
		return initServices()
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if telemetryService != nil {
			telemetryService.Close()
		}
		if dataStore != nil {
			return dataStore.Close()
		}
		return nil
	},
	// Default to TUI when no subcommand is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.fluid/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", true, "output JSON (default true)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(ipCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(sshInjectCmd)
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(vmsCmd)
	rootCmd.AddCommand(hostsCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(playbooksCmd)
	rootCmd.AddCommand(tuiCmd)
}

func initServices() error {
	var err error

	// Determine config path
	configPath := cfgFile
	if configPath == "" {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".fluid", "config.yaml")
	}

	// Ensure config directory and file exist with defaults
	if err := ensureConfigFile(configPath); err != nil {
		return fmt.Errorf("ensure config: %w", err)
	}

	// Load config
	cfg, err = config.LoadWithEnvOverride(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Ensure SSH CA exists - generate if missing
	created, err := sshca.EnsureSSHCA(cfg.SSH.CAKeyPath, cfg.SSH.CAPubPath, "fluid-ssh-ca")
	if err != nil {
		return fmt.Errorf("ensure SSH CA: %w", err)
	}
	if created {
		// Log that we created the CA (in JSON format for agent consumption)
		fmt.Fprintf(os.Stderr, `{"event":"ssh_ca_created","ca_key":"%s","ca_pub":"%s"}`+"\n",
			cfg.SSH.CAKeyPath, cfg.SSH.CAPubPath)
	}

	// Open SQLite store
	ctx := context.Background()
	dataStore, err = sqlite.New(ctx, store.Config{
		AutoMigrate: true,
	})
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}

	// Create and initialize SSH CA for key management
	ca, err := sshca.NewCA(sshca.Config{
		CAKeyPath:             cfg.SSH.CAKeyPath,
		CAPubKeyPath:          cfg.SSH.CAPubPath,
		WorkDir:               cfg.SSH.WorkDir,
		DefaultTTL:            cfg.SSH.CertTTL,
		MaxTTL:                cfg.SSH.MaxTTL,
		DefaultPrincipals:     []string{cfg.SSH.DefaultUser},
		EnforceKeyPermissions: true,
	})
	if err != nil {
		return fmt.Errorf("create SSH CA: %w", err)
	}
	if err := ca.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize SSH CA: %w", err)
	}

	// Create key manager for managed SSH credentials
	keyMgr, err := sshkeys.NewKeyManager(ca, sshkeys.Config{
		KeyDir:          cfg.SSH.KeyDir,
		CertificateTTL:  cfg.SSH.CertTTL,
		DefaultUsername: cfg.SSH.DefaultUser,
	}, slog.Default())
	if err != nil {
		return fmt.Errorf("create key manager: %w", err)
	}

	// Read SSH CA public key for injection into VMs via cloud-init
	sshCAPubKey := ""
	if pubKeyBytes, err := os.ReadFile(cfg.SSH.CAPubPath); err == nil {
		sshCAPubKey = strings.TrimSpace(string(pubKeyBytes))
	}

	// Create libvirt manager
	libvirtMgr = libvirt.NewVirshManager(libvirt.Config{
		LibvirtURI:         cfg.Libvirt.URI,
		BaseImageDir:       cfg.Libvirt.BaseImageDir,
		WorkDir:            cfg.Libvirt.WorkDir,
		SSHKeyInjectMethod: cfg.Libvirt.SSHKeyInjectMethod,
		SocketVMNetWrapper: cfg.Libvirt.SocketVMNetWrapper,
		SSHCAPubKey:        sshCAPubKey,
	}, slog.Default())

	// Initialize telemetry
	telemetryService, err = telemetry.NewService(cfg.Telemetry)
	if err != nil {
		// Fallback to no-op if telemetry fails
		telemetryService = telemetry.NewNoopService()
	}

	// Create VM service with key manager
	vmService = vm.NewService(libvirtMgr, dataStore, vm.Config{
		Network:            cfg.Libvirt.Network,
		DefaultVCPUs:       cfg.VM.DefaultVCPUs,
		DefaultMemoryMB:    cfg.VM.DefaultMemoryMB,
		CommandTimeout:     cfg.VM.CommandTimeout,
		IPDiscoveryTimeout: cfg.VM.IPDiscoveryTimeout,
		SSHProxyJump:       cfg.SSH.ProxyJump,
	}, vm.WithKeyManager(keyMgr), vm.WithTelemetry(telemetryService))

	return nil
}

// ensureConfigFile creates a default config file if it doesn't exist.
func ensureConfigFile(configPath string) error {
	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil // File exists
	}

	// Create config directory
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	// Write default config
	defaultCfg := `# Fluid CLI Configuration
# Auto-generated on first run

libvirt:
  uri: qemu:///system  # or qemu+ssh://user@host/system
  network: default
  base_image_dir: /var/lib/libvirt/images/base
  work_dir: /var/lib/libvirt/images/sandboxes
  ssh_key_inject_method: virt-customize

vm:
  default_vcpus: 2
  default_memory_mb: 2048
  command_timeout: 5m
  ip_discovery_timeout: 2m

ssh:
  proxy_jump: ""  # Optional: user@jumphost for isolated networks
  default_user: sandbox
  # SSH CA paths are auto-configured to ~/.fluid/ssh-ca/
`

	if err := os.WriteFile(configPath, []byte(defaultCfg), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// --- Init Command ---

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize fluid configuration",
	Long:  `Creates default config file at ~/.fluid/config.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		configDir := filepath.Join(home, ".fluid")
		configPath := filepath.Join(configDir, "config.yaml")

		// Check if config already exists
		if _, err := os.Stat(configPath); err == nil {
			output(map[string]any{
				"status":  "exists",
				"path":    configPath,
				"message": "Config file already exists",
			})
			return nil
		}

		// Create directory
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}

		// Write default config
		defaultCfg := `# Fluid CLI Configuration

libvirt:
  uri: qemu:///system  # or qemu+ssh://user@host/system
  network: default
  base_image_dir: /var/lib/libvirt/images/base
  work_dir: /var/lib/libvirt/images/sandboxes
  ssh_key_inject_method: virt-customize

vm:
  default_vcpus: 2
  default_memory_mb: 2048
  command_timeout: 5m
  ip_discovery_timeout: 2m

ssh:
  proxy_jump: ""  # Optional: user@jumphost for isolated networks
  default_user: sandbox
`

		if err := os.WriteFile(configPath, []byte(defaultCfg), 0o644); err != nil {
			return fmt.Errorf("write config: %w", err)
		}

		output(map[string]any{
			"status": "created",
			"path":   configPath,
		})
		return nil
	},
}

// --- Create Command ---

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new sandbox",
	Long:  `Create a new sandbox VM by cloning from a source VM`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceVM, _ := cmd.Flags().GetString("source-vm")
		agentID, _ := cmd.Flags().GetString("agent-id")
		cpu, _ := cmd.Flags().GetInt("cpu")
		memory, _ := cmd.Flags().GetInt("memory")
		autoStart, _ := cmd.Flags().GetBool("auto-start")
		waitIP, _ := cmd.Flags().GetBool("wait-ip")

		if sourceVM == "" {
			return fmt.Errorf("--source-vm is required")
		}
		if agentID == "" {
			agentID = "cli-agent"
		}

		ctx := context.Background()

		sb, ip, err := vmService.CreateSandbox(ctx, sourceVM, agentID, "", cpu, memory, nil, autoStart, waitIP)
		if err != nil {
			return err
		}

		result := map[string]interface{}{
			"sandbox_id": sb.ID,
			"name":       sb.SandboxName,
			"state":      sb.State,
		}
		if ip != "" {
			result["ip"] = ip
		}

		output(result)
		return nil
	},
}

func init() {
	createCmd.Flags().String("source-vm", "", "Source VM to clone from (required)")
	createCmd.Flags().String("agent-id", "", "Agent ID (default: cli-agent)")
	createCmd.Flags().Int("cpu", 0, "Number of vCPUs (default from config)")
	createCmd.Flags().Int("memory", 0, "Memory in MB (default from config)")
	createCmd.Flags().Bool("auto-start", true, "Auto-start the VM after creation")
	createCmd.Flags().Bool("wait-ip", true, "Wait for IP address discovery")
	createCmd.Flags().Bool("skip-validation", false, "Skip pre-flight validation checks")
}

// --- List Command ---

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List sandboxes",
	Long:  `List all sandboxes with optional filtering`,
	RunE: func(cmd *cobra.Command, args []string) error {
		state, _ := cmd.Flags().GetString("state")

		ctx := context.Background()
		filter := store.SandboxFilter{}
		if state != "" {
			s := store.SandboxState(strings.ToUpper(state))
			filter.State = &s
		}

		sandboxes, err := vmService.GetSandboxes(ctx, filter, nil)
		if err != nil {
			return err
		}

		result := make([]map[string]any, 0, len(sandboxes))
		for _, sb := range sandboxes {
			item := map[string]any{
				"sandbox_id": sb.ID,
				"name":       sb.SandboxName,
				"state":      sb.State,
				"base_image": sb.BaseImage,
				"created_at": sb.CreatedAt.Format(time.RFC3339),
			}
			if sb.IPAddress != nil {
				item["ip"] = *sb.IPAddress
			}
			result = append(result, item)
		}

		output(map[string]any{
			"sandboxes": result,
			"count":     len(result),
		})
		return nil
	},
}

func init() {
	listCmd.Flags().String("state", "", "Filter by state (CREATED, RUNNING, STOPPED, etc.)")
}

// --- Get Command ---

var getCmd = &cobra.Command{
	Use:   "get <sandbox-id>",
	Short: "Get sandbox details",
	Long:  `Get detailed information about a specific sandbox`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		sb, err := vmService.GetSandbox(ctx, args[0])
		if err != nil {
			return err
		}

		result := map[string]any{
			"sandbox_id": sb.ID,
			"name":       sb.SandboxName,
			"state":      sb.State,
			"base_image": sb.BaseImage,
			"network":    sb.Network,
			"agent_id":   sb.AgentID,
			"job_id":     sb.JobID,
			"created_at": sb.CreatedAt.Format(time.RFC3339),
			"updated_at": sb.UpdatedAt.Format(time.RFC3339),
		}
		if sb.IPAddress != nil {
			result["ip"] = *sb.IPAddress
		}

		output(result)
		return nil
	},
}

// --- Destroy Command ---

var destroyCmd = &cobra.Command{
	Use:   "destroy <sandbox-id>",
	Short: "Destroy a sandbox",
	Long:  `Completely destroy a sandbox VM and remove its storage`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		_, err := vmService.DestroySandbox(ctx, args[0])
		if err != nil {
			return err
		}

		output(map[string]interface{}{
			"destroyed":  true,
			"sandbox_id": args[0],
		})
		return nil
	},
}

// --- Start Command ---

var startCmd = &cobra.Command{
	Use:   "start <sandbox-id>",
	Short: "Start a sandbox",
	Long:  `Start a stopped sandbox VM`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		waitIP, _ := cmd.Flags().GetBool("wait-ip")

		ctx := context.Background()
		ip, err := vmService.StartSandbox(ctx, args[0], waitIP)
		if err != nil {
			return err
		}

		result := map[string]interface{}{
			"started":    true,
			"sandbox_id": args[0],
		}
		if ip != "" {
			result["ip"] = ip
		}

		output(result)
		return nil
	},
}

func init() {
	startCmd.Flags().Bool("wait-ip", true, "Wait for IP address discovery")
}

// --- Stop Command ---

var stopCmd = &cobra.Command{
	Use:   "stop <sandbox-id>",
	Short: "Stop a sandbox",
	Long:  `Stop a running sandbox VM`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		ctx := context.Background()
		err := vmService.StopSandbox(ctx, args[0], force)
		if err != nil {
			return err
		}

		output(map[string]interface{}{
			"stopped":    true,
			"sandbox_id": args[0],
		})
		return nil
	},
}

func init() {
	stopCmd.Flags().Bool("force", false, "Force stop (hard shutdown)")
}

// --- IP Command ---

var ipCmd = &cobra.Command{
	Use:   "ip <sandbox-id>",
	Short: "Discover IP address",
	Long:  `Discover or rediscover the IP address for a sandbox`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		ip, err := vmService.DiscoverIP(ctx, args[0])
		if err != nil {
			return err
		}

		output(map[string]interface{}{
			"sandbox_id": args[0],
			"ip":         ip,
		})
		return nil
	},
}

// --- Run Command ---

var runCmd = &cobra.Command{
	Use:   "run <sandbox-id> <command>",
	Short: "Run a command in a sandbox",
	Long:  `Execute a command inside the sandbox via SSH`,
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sandboxID := args[0]
		command := strings.Join(args[1:], " ")

		user, _ := cmd.Flags().GetString("user")
		key, _ := cmd.Flags().GetString("key")
		timeout, _ := cmd.Flags().GetDuration("timeout")

		if user == "" {
			user = cfg.SSH.DefaultUser
		}

		ctx := context.Background()
		result, err := vmService.RunCommand(ctx, sandboxID, user, key, command, timeout, nil)
		if err != nil {
			// Still return partial result if available
			if result != nil {
				output(map[string]interface{}{
					"sandbox_id": sandboxID,
					"exit_code":  result.ExitCode,
					"stdout":     result.Stdout,
					"stderr":     result.Stderr,
					"error":      err.Error(),
				})
				return nil
			}
			return err
		}

		output(map[string]interface{}{
			"sandbox_id": sandboxID,
			"exit_code":  result.ExitCode,
			"stdout":     result.Stdout,
			"stderr":     result.Stderr,
		})
		return nil
	},
}

func init() {
	runCmd.Flags().String("user", "", "SSH user (default from config)")
	runCmd.Flags().String("key", "", "SSH private key path")
	runCmd.Flags().Duration("timeout", 0, "Command timeout (default from config)")
}

// --- SSH Inject Command ---

var sshInjectCmd = &cobra.Command{
	Use:   "ssh-inject <sandbox-id>",
	Short: "Inject SSH public key",
	Long:  `Inject an SSH public key into the sandbox for the specified user`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sandboxID := args[0]
		pubkey, _ := cmd.Flags().GetString("pubkey")
		user, _ := cmd.Flags().GetString("user")

		if pubkey == "" {
			return fmt.Errorf("--pubkey is required")
		}
		if user == "" {
			user = cfg.SSH.DefaultUser
		}

		ctx := context.Background()
		err := vmService.InjectSSHKey(ctx, sandboxID, user, pubkey)
		if err != nil {
			return err
		}

		output(map[string]interface{}{
			"sandbox_id": sandboxID,
			"user":       user,
			"injected":   true,
		})
		return nil
	},
}

func init() {
	sshInjectCmd.Flags().String("pubkey", "", "SSH public key to inject (required)")
	sshInjectCmd.Flags().String("user", "", "Target user (default from config)")
}

// --- Snapshot Command ---

var snapshotCmd = &cobra.Command{
	Use:   "snapshot <sandbox-id>",
	Short: "Create a snapshot",
	Long:  `Create a snapshot of the current sandbox state`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sandboxID := args[0]
		name, _ := cmd.Flags().GetString("name")
		external, _ := cmd.Flags().GetBool("external")

		if name == "" {
			name = fmt.Sprintf("snap-%d", time.Now().Unix())
		}

		ctx := context.Background()
		snap, err := vmService.CreateSnapshot(ctx, sandboxID, name, external)
		if err != nil {
			return err
		}

		output(map[string]interface{}{
			"snapshot_id": snap.ID,
			"sandbox_id":  sandboxID,
			"name":        snap.Name,
			"kind":        snap.Kind,
		})
		return nil
	},
}

func init() {
	snapshotCmd.Flags().String("name", "", "Snapshot name (auto-generated if empty)")
	snapshotCmd.Flags().Bool("external", false, "Create external snapshot")
}

// --- Diff Command ---

var diffCmd = &cobra.Command{
	Use:   "diff <sandbox-id>",
	Short: "Compare snapshots",
	Long:  `Compute differences between two snapshots`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sandboxID := args[0]
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")

		if from == "" || to == "" {
			return fmt.Errorf("--from and --to are required")
		}

		ctx := context.Background()
		diff, err := vmService.DiffSnapshots(ctx, sandboxID, from, to)
		if err != nil {
			return err
		}

		output(map[string]interface{}{
			"diff_id":        diff.ID,
			"sandbox_id":     sandboxID,
			"from_snapshot":  diff.FromSnapshot,
			"to_snapshot":    diff.ToSnapshot,
			"files_added":    diff.DiffJSON.FilesAdded,
			"files_modified": diff.DiffJSON.FilesModified,
			"files_removed":  diff.DiffJSON.FilesRemoved,
		})
		return nil
	},
}

func init() {
	diffCmd.Flags().String("from", "", "Source snapshot name (required)")
	diffCmd.Flags().String("to", "", "Target snapshot name (required)")
}

// --- VMs Command ---

var vmsCmd = &cobra.Command{
	Use:   "vms",
	Short: "List available VMs",
	Long:  `List all VMs available in libvirt (potential source VMs for cloning)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Load config to check for hosts
		configPath := cfgFile
		if configPath == "" {
			home, _ := os.UserHomeDir()
			configPath = filepath.Join(home, ".fluid", "config.yaml")
		}

		loadedCfg, err := config.LoadWithEnvOverride(configPath)
		if err != nil {
			// If no config, fall back to local virsh
			vms, err := listVMsViaVirsh(ctx)
			if err != nil {
				return err
			}
			output(map[string]any{
				"vms":   vms,
				"count": len(vms),
			})
			return nil
		}

		// If hosts are configured, query remote hosts
		if len(loadedCfg.Hosts) > 0 {
			multiHostMgr := libvirt.NewMultiHostDomainManager(loadedCfg.Hosts, slog.Default())
			result, err := multiHostMgr.ListDomains(ctx)
			if err != nil {
				return fmt.Errorf("query hosts: %w", err)
			}

			vms := make([]map[string]any, 0, len(result.Domains))
			for _, d := range result.Domains {
				vms = append(vms, map[string]any{
					"name":  d.Name,
					"state": d.State.String(),
					"host":  d.HostName,
				})
			}

			resp := map[string]any{
				"vms":   vms,
				"count": len(vms),
			}

			// Include host errors if any
			if len(result.HostErrors) > 0 {
				errors := make([]map[string]any, 0, len(result.HostErrors))
				for _, e := range result.HostErrors {
					errors = append(errors, map[string]any{
						"host":  e.HostName,
						"error": e.Error,
					})
				}
				resp["host_errors"] = errors
			}

			output(resp)
			return nil
		}

		// No hosts configured, use local virsh
		vms, err := listVMsViaVirsh(ctx)
		if err != nil {
			return err
		}

		output(map[string]any{
			"vms":   vms,
			"count": len(vms),
		})
		return nil
	},
}

func listVMsViaVirsh(ctx context.Context) ([]map[string]any, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "virsh", "list", "--all", "--name")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("virsh list: %w: %s", err, stderr.String())
	}

	result := make([]map[string]any, 0)
	for _, name := range strings.Split(stdout.String(), "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		vmInfo := map[string]any{
			"name": name,
		}

		// Get additional info about each VM if libvirtMgr is available
		if libvirtMgr != nil {
			// Get VM state
			state, err := libvirtMgr.GetVMState(ctx, name)
			if err == nil {
				vmInfo["state"] = string(state)
			}

			// Get IP address (only for running VMs)
			if state == libvirt.VMStateRunning {
				ip, mac, err := libvirtMgr.GetIPAddress(ctx, name, 1*time.Second)
				if err == nil && ip != "" {
					vmInfo["ip"] = ip
					vmInfo["mac"] = mac
				}
			}
		}

		result = append(result, vmInfo)
	}

	return result, nil
}

// --- Validate Command ---

var validateCmd = &cobra.Command{
	Use:   "validate <source-vm>",
	Short: "Validate a source VM and host resources",
	Long: `Run pre-flight validation checks on a source VM before creating sandboxes.

This command checks:
- Source VM state (running/shut off)
- Network interface configuration
- MAC address assignment
- IP address (if VM is running)
- Host memory availability
- Host disk space`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceVM := args[0]
		memory, _ := cmd.Flags().GetInt("memory")

		ctx := context.Background()

		result := map[string]any{
			"source_vm": sourceVM,
			"valid":     true,
			"warnings":  []string{},
			"errors":    []string{},
		}

		var allWarnings []string
		var allErrors []string

		// Validate source VM
		vmValidation, err := libvirtMgr.ValidateSourceVM(ctx, sourceVM)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Failed to validate source VM: %v", err))
			result["valid"] = false
		} else {
			result["vm_state"] = vmValidation.State
			result["has_network"] = vmValidation.HasNetwork
			if vmValidation.MACAddress != "" {
				result["mac_address"] = vmValidation.MACAddress
			}
			if vmValidation.IPAddress != "" {
				result["ip_address"] = vmValidation.IPAddress
			}
			if !vmValidation.Valid {
				result["valid"] = false
			}
			allWarnings = append(allWarnings, vmValidation.Warnings...)
			allErrors = append(allErrors, vmValidation.Errors...)
		}

		// Check host resources
		memoryToCheck := memory
		if memoryToCheck <= 0 {
			memoryToCheck = cfg.VM.DefaultMemoryMB
		}
		// Use default CPU count if not specified (for validation purposes)
		cpuToCheck := cfg.VM.DefaultVCPUs
		resourceCheck, err := libvirtMgr.CheckHostResources(ctx, cpuToCheck, memoryToCheck)
		if err != nil {
			allWarnings = append(allWarnings, fmt.Sprintf("Failed to check host resources: %v", err))
		} else {
			result["host_memory_total_mb"] = resourceCheck.TotalMemoryMB
			result["host_memory_available_mb"] = resourceCheck.AvailableMemoryMB
			result["host_cpus_available"] = resourceCheck.AvailableCPUs
			result["host_disk_available_mb"] = resourceCheck.AvailableDiskMB
			result["required_memory_mb"] = resourceCheck.RequiredMemoryMB
			result["required_cpus"] = resourceCheck.RequiredCPUs
			if !resourceCheck.Valid {
				result["valid"] = false
			}
			allWarnings = append(allWarnings, resourceCheck.Warnings...)
			allErrors = append(allErrors, resourceCheck.Errors...)
		}

		result["warnings"] = allWarnings
		result["errors"] = allErrors

		output(result)
		return nil
	},
}

func init() {
	validateCmd.Flags().Int("memory", 0, "Memory in MB to check for (default from config)")
}

// --- Hosts Command ---

var hostsCmd = &cobra.Command{
	Use:   "hosts",
	Short: "List configured remote hosts",
	Long:  `List all remote libvirt hosts configured in the config file`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		configPath := cfgFile
		if configPath == "" {
			home, _ := os.UserHomeDir()
			configPath = filepath.Join(home, ".fluid", "config.yaml")
		}

		loadedCfg, err := config.LoadWithEnvOverride(configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if len(loadedCfg.Hosts) == 0 {
			output(map[string]any{
				"hosts":   []map[string]any{},
				"count":   0,
				"message": "No remote hosts configured. Add hosts to your config file.",
			})
			return nil
		}

		hosts := make([]map[string]any, 0, len(loadedCfg.Hosts))
		for _, h := range loadedCfg.Hosts {
			host := map[string]any{
				"name":    h.Name,
				"address": h.Address,
			}
			if h.SSHUser != "" {
				host["ssh_user"] = h.SSHUser
			}
			if h.SSHPort != 0 {
				host["ssh_port"] = h.SSHPort
			}
			hosts = append(hosts, host)
		}

		output(map[string]any{
			"hosts": hosts,
			"count": len(hosts),
		})
		return nil
	},
}

// --- Version Command ---

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		output(map[string]any{
			"version": "0.1.0",
			"name":    "fluid",
		})
		return nil
	},
}

// --- Playbooks Command ---

var playbooksCmd = &cobra.Command{
	Use:   "playbooks",
	Short: "List generated Ansible playbooks",
	Long:  `List all generated Ansible playbooks and provide links to open them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		playbooks, err := dataStore.ListPlaybooks(ctx, nil)
		if err != nil {
			return err
		}

		if outputJSON {
			type playbookOutput struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Path      string `json:"path"`
				CreatedAt string `json:"created_at"`
			}

			results := make([]playbookOutput, 0, len(playbooks))
			for _, pb := range playbooks {
				path := ""
				if pb.FilePath != nil && *pb.FilePath != "" {
					path = *pb.FilePath
				} else {
					path = filepath.Join(cfg.Ansible.PlaybooksDir, pb.Name+".yml")
				}
				results = append(results, playbookOutput{
					ID:        pb.ID,
					Name:      pb.Name,
					Path:      path,
					CreatedAt: pb.CreatedAt.Format(time.RFC3339),
				})
			}
			output(map[string]any{
				"playbooks": results,
				"count":     len(results),
			})
			return nil
		}

		if len(playbooks) == 0 {
			fmt.Println("No playbooks found.")
			return nil
		}

		fmt.Printf("Found %d playbook(s):\n\n", len(playbooks))
		for _, pb := range playbooks {
			path := ""
			if pb.FilePath != nil && *pb.FilePath != "" {
				path = *pb.FilePath
			} else {
				path = filepath.Join(cfg.Ansible.PlaybooksDir, pb.Name+".yml")
			}

			absPath, _ := filepath.Abs(path)
			// OSC 8 hyperlink
			link := fmt.Sprintf("\033]8;;file://%s\033\\%s\033]8;;\033\\", absPath, path)

			fmt.Printf("- %s: %s\n", pb.Name, link)
		}
		return nil
	},
}

// --- TUI Command ---

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive TUI",
	Long:  `Launch an interactive terminal UI for managing sandboxes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

// runTUI launches the interactive TUI
func runTUI() error {
	// Get config path
	configPath := cfgFile
	if configPath == "" {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".fluid", "config.yaml")
	}

	// Load config directly here to ensure hosts are loaded
	var err error
	cfg, err = tui.EnsureConfigExists(configPath)
	if err != nil {
		return fmt.Errorf("ensure config: %w", err)
	}

	// Check if onboarding is needed (first run)
	if !cfg.OnboardingComplete {
		// Run onboarding wizard
		updatedCfg, err := tui.RunOnboarding(cfg, configPath)
		if err != nil {
			return fmt.Errorf("onboarding: %w", err)
		}
		cfg = updatedCfg

		// Mark onboarding as complete and save config
		cfg.OnboardingComplete = true
		if err := cfg.Save(configPath); err != nil {
			// Non-fatal: continue even if we can't save the flag
			fmt.Fprintf(os.Stderr, "Warning: could not save onboarding status: %v\n", err)
		}
	}

	// Create a silent logger for TUI mode to prevent stdout corruption
	// slog output would corrupt the alternate screen buffer used by the TUI
	silentLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Initialize services with the loaded config and silent logger
	if err := initServicesWithConfigAndLogger(cfg, silentLogger); err != nil {
		return fmt.Errorf("init services: %w", err)
	}

	agent := tui.NewFluidAgent(cfg, dataStore, vmService, libvirtMgr, telemetryService)

	// Ensure cleanup runs when the TUI exits (user quits or Ctrl+C)
	defer func() {
		if agent.CreatedSandboxCount() > 0 {
			fmt.Printf("\nCleaning up %d session sandbox(es)...\n", agent.CreatedSandboxCount())
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := agent.Cleanup(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cleanup failed: %v\n", err)
			} else {
				fmt.Println("Cleanup complete.")
			}
		}
	}()

	model := tui.NewModel("fluid", "local", "vm-agent", agent, cfg, configPath)
	return tui.Run(model)
}

// initServicesWithConfigAndLogger initializes services with a pre-loaded config and custom logger
func initServicesWithConfigAndLogger(loadedCfg *config.Config, logger *slog.Logger) error {
	var err error

	cfg = loadedCfg

	// Ensure SSH CA exists - generate if missing
	// For TUI mode we don't log to stderr to avoid corrupting the display
	_, err = sshca.EnsureSSHCA(cfg.SSH.CAKeyPath, cfg.SSH.CAPubPath, "fluid-ssh-ca")
	if err != nil {
		return fmt.Errorf("ensure SSH CA: %w", err)
	}

	// Open SQLite store
	ctx := context.Background()
	dataStore, err = sqlite.New(ctx, store.Config{
		AutoMigrate: true,
	})
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}

	// Create and initialize SSH CA for key management
	ca, err := sshca.NewCA(sshca.Config{
		CAKeyPath:             cfg.SSH.CAKeyPath,
		CAPubKeyPath:          cfg.SSH.CAPubPath,
		WorkDir:               cfg.SSH.WorkDir,
		DefaultTTL:            cfg.SSH.CertTTL,
		MaxTTL:                cfg.SSH.MaxTTL,
		DefaultPrincipals:     []string{cfg.SSH.DefaultUser},
		EnforceKeyPermissions: true,
	})
	if err != nil {
		return fmt.Errorf("create SSH CA: %w", err)
	}
	if err := ca.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize SSH CA: %w", err)
	}

	// Create key manager for managed SSH credentials
	keyMgr, err := sshkeys.NewKeyManager(ca, sshkeys.Config{
		KeyDir:          cfg.SSH.KeyDir,
		CertificateTTL:  cfg.SSH.CertTTL,
		DefaultUsername: cfg.SSH.DefaultUser,
	}, logger)
	if err != nil {
		return fmt.Errorf("create key manager: %w", err)
	}

	// Read SSH CA public key for injection into VMs via cloud-init
	sshCAPubKey := ""
	if pubKeyBytes, err := os.ReadFile(cfg.SSH.CAPubPath); err == nil {
		sshCAPubKey = strings.TrimSpace(string(pubKeyBytes))
	}

	// Create libvirt config (shared between local and remote managers)
	virshCfg := libvirt.Config{
		LibvirtURI:         cfg.Libvirt.URI,
		BaseImageDir:       cfg.Libvirt.BaseImageDir,
		WorkDir:            cfg.Libvirt.WorkDir,
		SSHKeyInjectMethod: cfg.Libvirt.SSHKeyInjectMethod,
		SocketVMNetWrapper: cfg.Libvirt.SocketVMNetWrapper,
		DefaultNetwork:     cfg.Libvirt.Network,
		DefaultVCPUs:       cfg.VM.DefaultVCPUs,
		DefaultMemoryMB:    cfg.VM.DefaultMemoryMB,
		SSHCAPubKey:        sshCAPubKey,
	}

	// Create libvirt manager
	libvirtMgr = libvirt.NewVirshManager(virshCfg, logger)

	// Initialize telemetry
	telemetryService, err = telemetry.NewService(cfg.Telemetry)
	if err != nil {
		// Fallback to no-op if telemetry fails
		telemetryService = telemetry.NewNoopService()
	}

	// Create VM service with virsh config for remote host support
	vmService = vm.NewService(libvirtMgr, dataStore, vm.Config{
		Network:            cfg.Libvirt.Network,
		DefaultVCPUs:       cfg.VM.DefaultVCPUs,
		DefaultMemoryMB:    cfg.VM.DefaultMemoryMB,
		CommandTimeout:     cfg.VM.CommandTimeout,
		IPDiscoveryTimeout: cfg.VM.IPDiscoveryTimeout,
		SSHProxyJump:       cfg.SSH.ProxyJump,
	}, vm.WithVirshConfig(virshCfg), vm.WithLogger(logger), vm.WithKeyManager(keyMgr), vm.WithTelemetry(telemetryService))

	return nil
}

// --- Output Helpers ---

func output(v any) {
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(v)
	} else {
		// Human-readable output
		data, _ := yaml.Marshal(v)
		fmt.Print(string(data))
	}
}

func outputError(err error) {
	v := map[string]any{
		"error": err.Error(),
	}
	if outputJSON {
		enc := json.NewEncoder(os.Stderr)
		_ = enc.Encode(v)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	}
}
