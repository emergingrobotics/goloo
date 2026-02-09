package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/emergingrobotics/goloo/internal/cloudinit"
	"github.com/emergingrobotics/goloo/internal/config"
	"github.com/emergingrobotics/goloo/internal/provider"
	awsprovider "github.com/emergingrobotics/goloo/internal/provider/aws"
	"github.com/emergingrobotics/goloo/internal/provider/multipass"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

type Command struct {
	Action        string
	VMName        string
	ProviderFlag  string
	ConfigPath    string
	CloudInitPath string
}

func run(args []string) error {
	command, err := ParseArgs(args)
	if err != nil {
		return err
	}

	ctx := context.Background()

	switch command.Action {
	case "version":
		fmt.Println(version)
		return nil
	case "help":
		printUsage()
		return nil
	case "create":
		return cmdCreate(ctx, command)
	case "delete":
		return cmdDelete(ctx, command)
	case "list":
		return cmdList(ctx, command)
	case "ssh":
		return cmdSSH(ctx, command)
	case "status":
		return cmdStatus(ctx, command)
	case "stop":
		return cmdStop(ctx, command)
	case "start":
		return cmdStart(ctx, command)
	case "dns-swap":
		return cmdDNSSwap(ctx, command)
	default:
		return fmt.Errorf("unknown command %q\nRun 'goloo help' for usage", command.Action)
	}
}

func ParseArgs(args []string) (*Command, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no command provided\n\nUsage: goloo <command> <name> [flags]\nCommands: create, delete, list, ssh, status, stop, start, dns swap\n\nRun 'goloo help' for details")
	}

	first := args[0]
	if first == "--version" || first == "-v" {
		return &Command{Action: "version"}, nil
	}
	if first == "--help" || first == "-h" || first == "help" {
		return &Command{Action: "help"}, nil
	}

	if isLegacyInvocation(args) {
		return parseLegacyArgs(args)
	}

	if strings.HasPrefix(first, "-") {
		return nil, fmt.Errorf("unknown flag %q\nRun 'goloo help' for usage", first)
	}

	command := &Command{Action: first}
	remaining := args[1:]

	if command.Action == "dns" {
		if len(remaining) == 0 {
			return nil, fmt.Errorf("usage: goloo dns swap <name>")
		}
		if remaining[0] != "swap" {
			return nil, fmt.Errorf("unknown dns subcommand %q: use 'goloo dns swap <name>'", remaining[0])
		}
		command.Action = "dns-swap"
		remaining = remaining[1:]
	}

	if command.Action == "list" {
		for _, arg := range remaining {
			switch arg {
			case "--aws":
				command.ProviderFlag = "aws"
			case "--local":
				command.ProviderFlag = "local"
			default:
				return nil, fmt.Errorf("unknown flag %q for list command", arg)
			}
		}
		return command, nil
	}

	return parseNameAndFlags(command, remaining)
}

func isLegacyInvocation(args []string) bool {
	for _, arg := range args {
		if arg == "-c" || arg == "-d" {
			return true
		}
	}
	return false
}

func parseLegacyArgs(args []string) (*Command, error) {
	command := &Command{ProviderFlag: "aws"}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c":
			if command.Action == "delete" {
				return nil, fmt.Errorf("cannot use -c and -d together")
			}
			command.Action = "create"
		case "-d":
			if command.Action == "create" {
				return nil, fmt.Errorf("cannot use -c and -d together")
			}
			command.Action = "delete"
		case "-n":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("-n requires a VM name")
			}
			i++
			command.VMName = args[i]
		default:
			return nil, fmt.Errorf("unknown flag %q: legacy syntax is 'goloo -c -n <name>' or 'goloo -d -n <name>'", args[i])
		}
	}

	if command.Action == "" {
		return nil, fmt.Errorf("missing action flag: use -c (create) or -d (delete)")
	}
	if command.VMName == "" {
		return nil, fmt.Errorf("-n <name> required")
	}

	return command, nil
}

func parseNameAndFlags(command *Command, remaining []string) (*Command, error) {
	for i := 0; i < len(remaining); i++ {
		arg := remaining[i]
		switch {
		case arg == "--aws":
			command.ProviderFlag = "aws"
		case arg == "--local":
			command.ProviderFlag = "local"
		case arg == "--config" || arg == "-f":
			if i+1 >= len(remaining) {
				return nil, fmt.Errorf("%s requires a path argument", arg)
			}
			i++
			command.ConfigPath = remaining[i]
		case arg == "--cloud-init":
			if i+1 >= len(remaining) {
				return nil, fmt.Errorf("--cloud-init requires a path argument")
			}
			i++
			command.CloudInitPath = remaining[i]
		case strings.HasPrefix(arg, "-"):
			return nil, fmt.Errorf("unknown flag %q\nRun 'goloo help' for usage", arg)
		default:
			if command.VMName != "" {
				return nil, fmt.Errorf("unexpected argument %q after VM name %q", arg, command.VMName)
			}
			command.VMName = arg
		}
	}

	if command.VMName == "" {
		return nil, fmt.Errorf("VM name required: goloo %s <name>", command.Action)
	}

	return command, nil
}

func DetectProvider(providerFlag string, configuration *config.Config) string {
	if providerFlag == "aws" {
		return "aws"
	}
	if providerFlag == "local" {
		return "multipass"
	}
	if configuration.VM.StackID != "" {
		return "aws"
	}
	if configuration.DNS != nil && configuration.DNS.Domain != "" {
		return "aws"
	}
	return "multipass"
}

func getProvider(providerName string, region string) (provider.VMProvider, error) {
	switch providerName {
	case "aws":
		return awsprovider.New(region), nil
	case "multipass":
		return multipass.New(), nil
	default:
		return nil, fmt.Errorf("unknown provider %q: use 'aws' or 'multipass'", providerName)
	}
}

func loadConfig(command *Command) (*config.Config, string, error) {
	name := command.VMName
	if command.ConfigPath != "" {
		name = command.ConfigPath
	}
	return config.Load(name)
}

func resolveCloudInitPath(command *Command, configuration *config.Config) string {
	if command.CloudInitPath != "" {
		return command.CloudInitPath
	}
	return configuration.VM.CloudInitFile
}

func cmdCreate(ctx context.Context, command *Command) error {
	configuration, configPath, err := loadConfig(command)
	if err != nil {
		return err
	}

	providerName := DetectProvider(command.ProviderFlag, configuration)
	vmProvider, err := getProvider(providerName, configuration.VM.Region)
	if err != nil {
		return err
	}

	cloudInitSource := resolveCloudInitPath(command, configuration)
	cloudInitPath := ""
	if cloudInitSource != "" {
		processedPath, err := cloudinit.Process(cloudInitSource, configuration.VM.Users, cloudinit.FetchGitHubKeys)
		if err != nil {
			return fmt.Errorf("cloud-init processing failed: %w", err)
		}
		defer os.Remove(processedPath)
		cloudInitPath = processedPath
	}

	if err := vmProvider.Create(ctx, configuration, cloudInitPath); err != nil {
		return err
	}

	if err := config.Save(configPath, configuration); err != nil {
		return fmt.Errorf("VM created but failed to save config: %w", err)
	}

	fmt.Printf("Created %s via %s\n", configuration.VM.Name, vmProvider.Name())
	if configuration.VM.PublicIP != "" {
		fmt.Printf("IP: %s\n", configuration.VM.PublicIP)
	}
	if configuration.DNS != nil && configuration.DNS.FQDN != "" {
		fmt.Printf("DNS: %s\n", configuration.DNS.FQDN)
	}
	if len(configuration.VM.Users) > 0 {
		target := configuration.VM.PublicIP
		if configuration.DNS != nil && configuration.DNS.FQDN != "" {
			target = configuration.DNS.FQDN
		}
		if target != "" {
			fmt.Printf("SSH: ssh %s@%s\n", configuration.VM.Users[0].Username, target)
		}
	}

	return nil
}

func cmdDelete(ctx context.Context, command *Command) error {
	configuration, configPath, err := loadConfig(command)
	if err != nil {
		return err
	}

	providerName := DetectProvider(command.ProviderFlag, configuration)
	vmProvider, err := getProvider(providerName, configuration.VM.Region)
	if err != nil {
		return err
	}

	if err := vmProvider.Delete(ctx, configuration); err != nil {
		return err
	}

	if err := config.Save(configPath, configuration); err != nil {
		return fmt.Errorf("VM deleted but failed to save config: %w", err)
	}

	fmt.Printf("Deleted %s\n", configuration.VM.Name)
	return nil
}

func cmdList(ctx context.Context, command *Command) error {
	providerName := "multipass"
	if command.ProviderFlag == "aws" {
		providerName = "aws"
	}

	vmProvider, err := getProvider(providerName, "us-east-1")
	if err != nil {
		return err
	}

	statuses, err := vmProvider.List(ctx)
	if err != nil {
		return err
	}

	if len(statuses) == 0 {
		fmt.Println("No VMs found")
		return nil
	}

	fmt.Printf("%-20s %-12s %-16s %s\n", "NAME", "STATE", "IP", "PROVIDER")
	for _, status := range statuses {
		ip := status.IP
		if ip == "" {
			ip = "-"
		}
		fmt.Printf("%-20s %-12s %-16s %s\n", status.Name, status.State, ip, status.Provider)
	}

	return nil
}

func cmdSSH(ctx context.Context, command *Command) error {
	configuration, _, err := loadConfig(command)
	if err != nil {
		return err
	}

	providerName := DetectProvider(command.ProviderFlag, configuration)
	vmProvider, err := getProvider(providerName, configuration.VM.Region)
	if err != nil {
		return err
	}

	return vmProvider.SSH(ctx, configuration)
}

func cmdStatus(ctx context.Context, command *Command) error {
	configuration, _, err := loadConfig(command)
	if err != nil {
		return err
	}

	providerName := DetectProvider(command.ProviderFlag, configuration)
	vmProvider, err := getProvider(providerName, configuration.VM.Region)
	if err != nil {
		return err
	}

	status, err := vmProvider.Status(ctx, configuration)
	if err != nil {
		return err
	}

	fmt.Printf("Name:     %s\n", status.Name)
	fmt.Printf("State:    %s\n", status.State)
	fmt.Printf("Provider: %s\n", status.Provider)
	if status.IP != "" {
		fmt.Printf("IP:       %s\n", status.IP)
	}

	return nil
}

func cmdStop(ctx context.Context, command *Command) error {
	configuration, _, err := loadConfig(command)
	if err != nil {
		return err
	}

	providerName := DetectProvider(command.ProviderFlag, configuration)
	vmProvider, err := getProvider(providerName, configuration.VM.Region)
	if err != nil {
		return err
	}

	if err := vmProvider.Stop(ctx, configuration); err != nil {
		return err
	}

	fmt.Printf("Stopped %s\n", configuration.VM.Name)
	return nil
}

func cmdStart(ctx context.Context, command *Command) error {
	configuration, _, err := loadConfig(command)
	if err != nil {
		return err
	}

	providerName := DetectProvider(command.ProviderFlag, configuration)
	vmProvider, err := getProvider(providerName, configuration.VM.Region)
	if err != nil {
		return err
	}

	if err := vmProvider.Start(ctx, configuration); err != nil {
		return err
	}

	fmt.Printf("Started %s\n", configuration.VM.Name)
	return nil
}

func cmdDNSSwap(ctx context.Context, command *Command) error {
	configuration, configPath, err := loadConfig(command)
	if err != nil {
		return err
	}

	awsProvider := awsprovider.New(configuration.VM.Region)
	if err := awsProvider.SwapDNS(ctx, configuration); err != nil {
		return err
	}

	if err := config.Save(configPath, configuration); err != nil {
		return fmt.Errorf("DNS swapped but failed to save config: %w", err)
	}

	fmt.Printf("DNS swapped: %s -> %s\n", configuration.DNS.FQDN, configuration.VM.PublicIP)
	return nil
}

func printUsage() {
	fmt.Println("goloo - Unified VM Provisioning")
	fmt.Println()
	fmt.Println("Usage: goloo <command> <name> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  create <name>       Create a VM")
	fmt.Println("  delete <name>       Delete a VM")
	fmt.Println("  list                List all VMs")
	fmt.Println("  ssh <name>          SSH into a VM")
	fmt.Println("  status <name>       Show VM status")
	fmt.Println("  stop <name>         Stop a VM")
	fmt.Println("  start <name>        Start a VM")
	fmt.Println("  dns swap <name>     Swap DNS to current VM IP")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --aws               Use AWS provider")
	fmt.Println("  --local             Use local Multipass provider")
	fmt.Println("  --config, -f PATH   Config file path")
	fmt.Println("  --cloud-init PATH   Cloud-init file path")
	fmt.Println("  --version, -v       Show version")
	fmt.Println("  --help, -h          Show this help")
	fmt.Println()
	fmt.Println("Legacy Flags (aws-ec2 compatibility):")
	fmt.Println("  -c -n <name>        Create AWS VM")
	fmt.Println("  -d -n <name>        Delete AWS VM")
	fmt.Println()
	fmt.Println("Provider Auto-Detection:")
	fmt.Println("  If no --aws or --local flag is given, the provider is detected from:")
	fmt.Println("  1. Existing stack_id in config -> AWS")
	fmt.Println("  2. DNS domain in config -> AWS")
	fmt.Println("  3. Default -> Multipass (local)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  goloo create devbox                  Create local VM")
	fmt.Println("  goloo create devbox --aws            Create AWS VM")
	fmt.Println("  goloo delete devbox                  Delete VM (auto-detects provider)")
	fmt.Println("  goloo ssh devbox                     SSH into VM")
	fmt.Println("  goloo dns swap devbox                Update DNS to current IP")
}
