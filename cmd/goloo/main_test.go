package main

import (
	"testing"

	"github.com/emergingrobotics/goloo/internal/config"
)

func TestParseArgsNoArgs(t *testing.T) {
	_, err := ParseArgs([]string{})
	if err == nil {
		t.Fatal("expected error for empty args")
	}
}

func TestParseArgsVersionFlag(t *testing.T) {
	command, err := ParseArgs([]string{"--version"})
	if err != nil {
		t.Fatalf("unexpected error for --version: %v", err)
	}
	if command.Action != "version" {
		t.Errorf("expected action 'version' for --version, got %q", command.Action)
	}
}

func TestParseArgsVerboseFlag(t *testing.T) {
	command, err := ParseArgs([]string{"create", "devbox", "-v"})
	if err != nil {
		t.Fatal(err)
	}
	if !command.Verbose {
		t.Error("expected Verbose=true for -v flag")
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
}

func TestParseArgsVerboseLongFlag(t *testing.T) {
	command, err := ParseArgs([]string{"create", "--verbose", "devbox"})
	if err != nil {
		t.Fatal(err)
	}
	if !command.Verbose {
		t.Error("expected Verbose=true for --verbose flag")
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
}

func TestParseArgsVerboseAloneIsError(t *testing.T) {
	_, err := ParseArgs([]string{"-v"})
	if err == nil {
		t.Fatal("expected error for -v with no command")
	}
}

func TestParseArgsHelpFlags(t *testing.T) {
	for _, flag := range []string{"--help", "-h", "help"} {
		command, err := ParseArgs([]string{flag})
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", flag, err)
		}
		if command.Action != "help" {
			t.Errorf("expected action 'help' for %s, got %q", flag, command.Action)
		}
	}
}

func TestParseArgsCreateWithName(t *testing.T) {
	command, err := ParseArgs([]string{"create", "devbox"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "create" {
		t.Errorf("expected action 'create', got %q", command.Action)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
	if command.ProviderFlag != "" {
		t.Errorf("expected empty ProviderFlag, got %q", command.ProviderFlag)
	}
}

func TestParseArgsCreateWithAWSFlag(t *testing.T) {
	command, err := ParseArgs([]string{"create", "devbox", "--aws"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "create" {
		t.Errorf("expected action 'create', got %q", command.Action)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
	if command.ProviderFlag != "aws" {
		t.Errorf("expected ProviderFlag 'aws', got %q", command.ProviderFlag)
	}
}

func TestParseArgsCreateWithLocalFlag(t *testing.T) {
	command, err := ParseArgs([]string{"create", "devbox", "--local"})
	if err != nil {
		t.Fatal(err)
	}
	if command.ProviderFlag != "local" {
		t.Errorf("expected ProviderFlag 'local', got %q", command.ProviderFlag)
	}
}

func TestParseArgsCreateWithConfigFlag(t *testing.T) {
	for _, flag := range []string{"--config", "-f"} {
		command, err := ParseArgs([]string{"create", "devbox", flag, "/path/to/config.json"})
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", flag, err)
		}
		if command.ConfigPath != "/path/to/config.json" {
			t.Errorf("expected ConfigPath '/path/to/config.json' for %s, got %q", flag, command.ConfigPath)
		}
	}
}

func TestParseArgsCreateWithCloudInitFlag(t *testing.T) {
	command, err := ParseArgs([]string{"create", "devbox", "--cloud-init", "cloud-init/dev.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if command.CloudInitPath != "cloud-init/dev.yaml" {
		t.Errorf("expected CloudInitPath 'cloud-init/dev.yaml', got %q", command.CloudInitPath)
	}
}

func TestParseArgsCreateWithMultipleFlags(t *testing.T) {
	command, err := ParseArgs([]string{"create", "devbox", "--aws", "--config", "stacks/prod.json", "--cloud-init", "ci.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if command.ProviderFlag != "aws" {
		t.Errorf("expected ProviderFlag 'aws', got %q", command.ProviderFlag)
	}
	if command.ConfigPath != "stacks/prod.json" {
		t.Errorf("expected ConfigPath 'stacks/prod.json', got %q", command.ConfigPath)
	}
	if command.CloudInitPath != "ci.yaml" {
		t.Errorf("expected CloudInitPath 'ci.yaml', got %q", command.CloudInitPath)
	}
}

func TestParseArgsDeleteWithName(t *testing.T) {
	command, err := ParseArgs([]string{"delete", "devbox"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "delete" {
		t.Errorf("expected action 'delete', got %q", command.Action)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
}

func TestParseArgsList(t *testing.T) {
	command, err := ParseArgs([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "list" {
		t.Errorf("expected action 'list', got %q", command.Action)
	}
	if command.VMName != "" {
		t.Errorf("expected empty VMName for list, got %q", command.VMName)
	}
}

func TestParseArgsListWithAWSFlag(t *testing.T) {
	command, err := ParseArgs([]string{"list", "--aws"})
	if err != nil {
		t.Fatal(err)
	}
	if command.ProviderFlag != "aws" {
		t.Errorf("expected ProviderFlag 'aws', got %q", command.ProviderFlag)
	}
}

func TestParseArgsListWithLocalFlag(t *testing.T) {
	command, err := ParseArgs([]string{"list", "--local"})
	if err != nil {
		t.Fatal(err)
	}
	if command.ProviderFlag != "local" {
		t.Errorf("expected ProviderFlag 'local', got %q", command.ProviderFlag)
	}
}

func TestParseArgsSSH(t *testing.T) {
	command, err := ParseArgs([]string{"ssh", "devbox"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "ssh" {
		t.Errorf("expected action 'ssh', got %q", command.Action)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
}

func TestParseArgsStatus(t *testing.T) {
	command, err := ParseArgs([]string{"status", "devbox"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "status" {
		t.Errorf("expected action 'status', got %q", command.Action)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
}

func TestParseArgsStopAndStart(t *testing.T) {
	for _, action := range []string{"stop", "start"} {
		command, err := ParseArgs([]string{action, "devbox"})
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", action, err)
		}
		if command.Action != action {
			t.Errorf("expected action %q, got %q", action, command.Action)
		}
		if command.VMName != "devbox" {
			t.Errorf("expected VMName 'devbox' for %s, got %q", action, command.VMName)
		}
	}
}

func TestParseArgsDNSSwap(t *testing.T) {
	command, err := ParseArgs([]string{"dns", "swap", "devbox"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "dns-swap" {
		t.Errorf("expected action 'dns-swap', got %q", command.Action)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
}

func TestParseArgsDNSWithoutSubcommand(t *testing.T) {
	_, err := ParseArgs([]string{"dns"})
	if err == nil {
		t.Fatal("expected error for dns without subcommand")
	}
}

func TestParseArgsDNSSwapWithoutName(t *testing.T) {
	_, err := ParseArgs([]string{"dns", "swap"})
	if err == nil {
		t.Fatal("expected error for dns swap without name")
	}
}

func TestParseArgsDNSUnknownSubcommand(t *testing.T) {
	_, err := ParseArgs([]string{"dns", "unknown", "devbox"})
	if err == nil {
		t.Fatal("expected error for unknown dns subcommand")
	}
}

func TestParseArgsLegacyCreate(t *testing.T) {
	command, err := ParseArgs([]string{"-c", "-n", "devbox"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "create" {
		t.Errorf("expected action 'create', got %q", command.Action)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
	if command.ProviderFlag != "aws" {
		t.Errorf("expected ProviderFlag 'aws' for legacy create, got %q", command.ProviderFlag)
	}
}

func TestParseArgsLegacyDelete(t *testing.T) {
	command, err := ParseArgs([]string{"-d", "-n", "devbox"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "delete" {
		t.Errorf("expected action 'delete', got %q", command.Action)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
	if command.ProviderFlag != "aws" {
		t.Errorf("expected ProviderFlag 'aws' for legacy delete, got %q", command.ProviderFlag)
	}
}

func TestParseArgsLegacyReversedOrder(t *testing.T) {
	command, err := ParseArgs([]string{"-n", "devbox", "-c"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "create" {
		t.Errorf("expected action 'create', got %q", command.Action)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
}

func TestParseArgsLegacyDeleteReversed(t *testing.T) {
	command, err := ParseArgs([]string{"-n", "devbox", "-d"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "delete" {
		t.Errorf("expected action 'delete', got %q", command.Action)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
}

func TestParseArgsLegacyWithoutName(t *testing.T) {
	_, err := ParseArgs([]string{"-c"})
	if err == nil {
		t.Fatal("expected error for -c without -n")
	}
}

func TestParseArgsLegacyConflictingFlags(t *testing.T) {
	_, err := ParseArgs([]string{"-c", "-d", "-n", "devbox"})
	if err == nil {
		t.Fatal("expected error for -c and -d together")
	}
}

func TestParseArgsLegacyNameMissingValue(t *testing.T) {
	_, err := ParseArgs([]string{"-c", "-n"})
	if err == nil {
		t.Fatal("expected error for -n without value")
	}
}

func TestParseArgsMissingNameForCommands(t *testing.T) {
	for _, action := range []string{"create", "delete", "ssh", "status", "stop", "start"} {
		_, err := ParseArgs([]string{action})
		if err == nil {
			t.Fatalf("expected error for %s without name", action)
		}
	}
}

func TestParseArgsUnknownFlag(t *testing.T) {
	_, err := ParseArgs([]string{"create", "devbox", "--unknown"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestParseArgsUnknownFirstFlag(t *testing.T) {
	_, err := ParseArgs([]string{"--unknown"})
	if err == nil {
		t.Fatal("expected error for unknown first flag")
	}
}

func TestParseArgsConfigFlagWithoutValue(t *testing.T) {
	_, err := ParseArgs([]string{"create", "devbox", "--config"})
	if err == nil {
		t.Fatal("expected error for --config without value")
	}
}

func TestParseArgsShortConfigFlagWithoutValue(t *testing.T) {
	_, err := ParseArgs([]string{"create", "devbox", "-f"})
	if err == nil {
		t.Fatal("expected error for -f without value")
	}
}

func TestParseArgsCloudInitFlagWithoutValue(t *testing.T) {
	_, err := ParseArgs([]string{"create", "devbox", "--cloud-init"})
	if err == nil {
		t.Fatal("expected error for --cloud-init without value")
	}
}

func TestParseArgsListWithUnknownFlag(t *testing.T) {
	_, err := ParseArgs([]string{"list", "--unknown"})
	if err == nil {
		t.Fatal("expected error for unknown flag on list")
	}
}

func TestParseArgsDNSSwapWithFlags(t *testing.T) {
	command, err := ParseArgs([]string{"dns", "swap", "devbox", "--config", "stacks/prod.json"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "dns-swap" {
		t.Errorf("expected action 'dns-swap', got %q", command.Action)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
	if command.ConfigPath != "stacks/prod.json" {
		t.Errorf("expected ConfigPath 'stacks/prod.json', got %q", command.ConfigPath)
	}
}

func TestParseArgsFlagsBeforeName(t *testing.T) {
	command, err := ParseArgs([]string{"create", "--aws", "devbox"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "create" {
		t.Errorf("expected action 'create', got %q", command.Action)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
	if command.ProviderFlag != "aws" {
		t.Errorf("expected ProviderFlag 'aws', got %q", command.ProviderFlag)
	}
}

func TestParseArgsConfigFlagBeforeName(t *testing.T) {
	command, err := ParseArgs([]string{"create", "--config", "stacks/prod.json", "devbox"})
	if err != nil {
		t.Fatal(err)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
	if command.ConfigPath != "stacks/prod.json" {
		t.Errorf("expected ConfigPath 'stacks/prod.json', got %q", command.ConfigPath)
	}
}

func TestParseArgsDuplicatePositionalArgs(t *testing.T) {
	_, err := ParseArgs([]string{"create", "devbox", "extra"})
	if err == nil {
		t.Fatal("expected error for duplicate positional args")
	}
}

func TestParseArgsMixedFlagPositions(t *testing.T) {
	command, err := ParseArgs([]string{"create", "--aws", "devbox", "--cloud-init", "ci.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
	if command.ProviderFlag != "aws" {
		t.Errorf("expected ProviderFlag 'aws', got %q", command.ProviderFlag)
	}
	if command.CloudInitPath != "ci.yaml" {
		t.Errorf("expected CloudInitPath 'ci.yaml', got %q", command.CloudInitPath)
	}
}

func TestDetectProviderAWSFlag(t *testing.T) {
	configuration := &config.Config{VM: &config.VMConfig{}}
	result := DetectProvider("aws", configuration)
	if result != "aws" {
		t.Errorf("expected 'aws', got %q", result)
	}
}

func TestDetectProviderLocalFlag(t *testing.T) {
	configuration := &config.Config{VM: &config.VMConfig{}}
	result := DetectProvider("local", configuration)
	if result != "multipass" {
		t.Errorf("expected 'multipass', got %q", result)
	}
}

func TestDetectProviderFromStackID(t *testing.T) {
	configuration := &config.Config{VM: &config.VMConfig{StackID: "arn:aws:cloudformation:us-east-1:123456789:stack/goloo-test/abc"}}
	result := DetectProvider("", configuration)
	if result != "aws" {
		t.Errorf("expected 'aws' from stack_id, got %q", result)
	}
}

func TestDetectProviderFromDNSDomain(t *testing.T) {
	configuration := &config.Config{
		VM:  &config.VMConfig{},
		DNS: &config.DNSConfig{Domain: "example.com"},
	}
	result := DetectProvider("", configuration)
	if result != "aws" {
		t.Errorf("expected 'aws' from dns domain, got %q", result)
	}
}

func TestDetectProviderDefault(t *testing.T) {
	configuration := &config.Config{VM: &config.VMConfig{}}
	result := DetectProvider("", configuration)
	if result != "multipass" {
		t.Errorf("expected 'multipass' as default, got %q", result)
	}
}

func TestDetectProviderFlagOverridesStackID(t *testing.T) {
	configuration := &config.Config{VM: &config.VMConfig{StackID: "arn:aws:cloudformation:..."}}
	result := DetectProvider("local", configuration)
	if result != "multipass" {
		t.Errorf("expected 'multipass' (flag override), got %q", result)
	}
}

func TestDetectProviderFlagOverridesDNS(t *testing.T) {
	configuration := &config.Config{
		VM:  &config.VMConfig{},
		DNS: &config.DNSConfig{Domain: "example.com"},
	}
	result := DetectProvider("local", configuration)
	if result != "multipass" {
		t.Errorf("expected 'multipass' (flag override), got %q", result)
	}
}

func TestDetectProviderStackIDBeforeDNS(t *testing.T) {
	configuration := &config.Config{
		VM:  &config.VMConfig{StackID: "arn:aws:cloudformation:..."},
		DNS: &config.DNSConfig{Domain: "example.com"},
	}
	result := DetectProvider("", configuration)
	if result != "aws" {
		t.Errorf("expected 'aws' from stack_id, got %q", result)
	}
}

func TestDetectProviderNilDNS(t *testing.T) {
	configuration := &config.Config{VM: &config.VMConfig{}}
	result := DetectProvider("", configuration)
	if result != "multipass" {
		t.Errorf("expected 'multipass' with nil DNS, got %q", result)
	}
}

func TestDetectProviderEmptyDNSDomain(t *testing.T) {
	configuration := &config.Config{
		VM:  &config.VMConfig{},
		DNS: &config.DNSConfig{},
	}
	result := DetectProvider("", configuration)
	if result != "multipass" {
		t.Errorf("expected 'multipass' with empty DNS domain, got %q", result)
	}
}
