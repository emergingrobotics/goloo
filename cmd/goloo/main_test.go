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

func TestParseArgsCreateWithFolderFlag(t *testing.T) {
	for _, flag := range []string{"--folder", "-f"} {
		command, err := ParseArgs([]string{"create", "devbox", flag, "/path/to/servers"})
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", flag, err)
		}
		if command.FolderPath != "/path/to/servers" {
			t.Errorf("expected FolderPath '/path/to/servers' for %s, got %q", flag, command.FolderPath)
		}
	}
}

func TestParseArgsCreateWithMultipleFlags(t *testing.T) {
	command, err := ParseArgs([]string{"create", "devbox", "--aws", "--folder", "/my/stacks"})
	if err != nil {
		t.Fatal(err)
	}
	if command.ProviderFlag != "aws" {
		t.Errorf("expected ProviderFlag 'aws', got %q", command.ProviderFlag)
	}
	if command.FolderPath != "/my/stacks" {
		t.Errorf("expected FolderPath '/my/stacks', got %q", command.FolderPath)
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

func TestParseArgsFolderFlagWithoutValue(t *testing.T) {
	_, err := ParseArgs([]string{"create", "devbox", "--folder"})
	if err == nil {
		t.Fatal("expected error for --folder without value")
	}
}

func TestParseArgsShortFolderFlagWithoutValue(t *testing.T) {
	_, err := ParseArgs([]string{"create", "devbox", "-f"})
	if err == nil {
		t.Fatal("expected error for -f without value")
	}
}

func TestParseArgsListWithUnknownFlag(t *testing.T) {
	_, err := ParseArgs([]string{"list", "--unknown"})
	if err == nil {
		t.Fatal("expected error for unknown flag on list")
	}
}

func TestParseArgsDNSSwapWithFlags(t *testing.T) {
	command, err := ParseArgs([]string{"dns", "swap", "devbox", "--folder", "/my/stacks"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != "dns-swap" {
		t.Errorf("expected action 'dns-swap', got %q", command.Action)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
	if command.FolderPath != "/my/stacks" {
		t.Errorf("expected FolderPath '/my/stacks', got %q", command.FolderPath)
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

func TestParseArgsFolderFlagBeforeName(t *testing.T) {
	command, err := ParseArgs([]string{"create", "--folder", "/my/stacks", "devbox"})
	if err != nil {
		t.Fatal(err)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
	if command.FolderPath != "/my/stacks" {
		t.Errorf("expected FolderPath '/my/stacks', got %q", command.FolderPath)
	}
}

func TestParseArgsDuplicatePositionalArgs(t *testing.T) {
	_, err := ParseArgs([]string{"create", "devbox", "extra"})
	if err == nil {
		t.Fatal("expected error for duplicate positional args")
	}
}

func TestParseArgsMixedFlagPositions(t *testing.T) {
	command, err := ParseArgs([]string{"create", "--aws", "devbox", "--folder", "/my/stacks"})
	if err != nil {
		t.Fatal(err)
	}
	if command.VMName != "devbox" {
		t.Errorf("expected VMName 'devbox', got %q", command.VMName)
	}
	if command.ProviderFlag != "aws" {
		t.Errorf("expected ProviderFlag 'aws', got %q", command.ProviderFlag)
	}
	if command.FolderPath != "/my/stacks" {
		t.Errorf("expected FolderPath '/my/stacks', got %q", command.FolderPath)
	}
}

func TestParseArgsUsersFlagSingleUser(t *testing.T) {
	command, err := ParseArgs([]string{"create", "devbox", "--users", "gherlein"})
	if err != nil {
		t.Fatal(err)
	}
	if len(command.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(command.Users))
	}
	if command.Users[0] != "gherlein" {
		t.Errorf("expected user 'gherlein', got %q", command.Users[0])
	}
}

func TestParseArgsUsersFlagCommaSeparated(t *testing.T) {
	command, err := ParseArgs([]string{"create", "devbox", "-u", "alice,bob,charlie"})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"alice", "bob", "charlie"}
	if len(command.Users) != len(expected) {
		t.Fatalf("expected %d users, got %d", len(expected), len(command.Users))
	}
	for i, name := range expected {
		if command.Users[i] != name {
			t.Errorf("expected user[%d] = %q, got %q", i, name, command.Users[i])
		}
	}
}

func TestParseArgsUsersFlagTrimsWhitespace(t *testing.T) {
	command, err := ParseArgs([]string{"create", "devbox", "-u", "alice , bob"})
	if err != nil {
		t.Fatal(err)
	}
	if len(command.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(command.Users))
	}
	if command.Users[0] != "alice" {
		t.Errorf("expected user[0] = 'alice', got %q", command.Users[0])
	}
	if command.Users[1] != "bob" {
		t.Errorf("expected user[1] = 'bob', got %q", command.Users[1])
	}
}

func TestParseArgsUsersFlagSkipsEmpty(t *testing.T) {
	command, err := ParseArgs([]string{"create", "devbox", "-u", "alice,,bob"})
	if err != nil {
		t.Fatal(err)
	}
	if len(command.Users) != 2 {
		t.Fatalf("expected 2 users (empty segments skipped), got %d", len(command.Users))
	}
}

func TestParseArgsUsersFlagWithoutValue(t *testing.T) {
	_, err := ParseArgs([]string{"create", "devbox", "--users"})
	if err == nil {
		t.Fatal("expected error for --users without value")
	}
}

func TestParseArgsShortUsersFlagWithoutValue(t *testing.T) {
	_, err := ParseArgs([]string{"create", "devbox", "-u"})
	if err == nil {
		t.Fatal("expected error for -u without value")
	}
}

func TestParseArgsUsersFlagWithOtherFlags(t *testing.T) {
	command, err := ParseArgs([]string{"create", "devbox", "--aws", "-u", "gherlein", "-f", "/my/stacks"})
	if err != nil {
		t.Fatal(err)
	}
	if command.ProviderFlag != "aws" {
		t.Errorf("expected ProviderFlag 'aws', got %q", command.ProviderFlag)
	}
	if len(command.Users) != 1 || command.Users[0] != "gherlein" {
		t.Errorf("expected users [gherlein], got %v", command.Users)
	}
	if command.FolderPath != "/my/stacks" {
		t.Errorf("expected FolderPath '/my/stacks', got %q", command.FolderPath)
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

func TestDetectProviderFromAWSState(t *testing.T) {
	configuration := &config.Config{
		VM:  &config.VMConfig{},
		AWS: &config.AWSState{StackID: "arn:aws:cloudformation:us-east-1:123456789:stack/goloo-test/abc"},
	}
	result := DetectProvider("", configuration)
	if result != "aws" {
		t.Errorf("expected 'aws' from AWS state, got %q", result)
	}
}

func TestDetectProviderDNSDomainDoesNotForceAWS(t *testing.T) {
	configuration := &config.Config{
		VM:  &config.VMConfig{},
		DNS: &config.DNSConfig{Domain: "example.com"},
	}
	result := DetectProvider("", configuration)
	if result != "multipass" {
		t.Errorf("expected 'multipass' (DNS alone should not force AWS), got %q", result)
	}
}

func TestDetectProviderDefault(t *testing.T) {
	configuration := &config.Config{VM: &config.VMConfig{}}
	result := DetectProvider("", configuration)
	if result != "multipass" {
		t.Errorf("expected 'multipass' as default, got %q", result)
	}
}

func TestDetectProviderFlagOverridesAWSState(t *testing.T) {
	configuration := &config.Config{
		VM:  &config.VMConfig{},
		AWS: &config.AWSState{StackID: "arn:aws:cloudformation:..."},
	}
	result := DetectProvider("local", configuration)
	if result != "multipass" {
		t.Errorf("expected 'multipass' (flag override), got %q", result)
	}
}

func TestDetectProviderAWSFlagWithDNS(t *testing.T) {
	configuration := &config.Config{
		VM:  &config.VMConfig{},
		DNS: &config.DNSConfig{Domain: "example.com"},
	}
	result := DetectProvider("aws", configuration)
	if result != "aws" {
		t.Errorf("expected 'aws' (explicit flag), got %q", result)
	}
}

func TestDetectProviderAWSStateBeforeDNS(t *testing.T) {
	configuration := &config.Config{
		VM:  &config.VMConfig{},
		DNS: &config.DNSConfig{Domain: "example.com"},
		AWS: &config.AWSState{StackID: "arn:aws:cloudformation:..."},
	}
	result := DetectProvider("", configuration)
	if result != "aws" {
		t.Errorf("expected 'aws' from AWS state, got %q", result)
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

func TestResolveStackFolderDefault(t *testing.T) {
	t.Setenv("GOLOO_STACK_FOLDER", "")
	command := &Command{}
	result := resolveStackFolder(command)
	if result != "stacks" {
		t.Errorf("expected 'stacks', got %q", result)
	}
}

func TestResolveStackFolderFromEnv(t *testing.T) {
	t.Setenv("GOLOO_STACK_FOLDER", "/home/user/my-servers")
	command := &Command{}
	result := resolveStackFolder(command)
	if result != "/home/user/my-servers" {
		t.Errorf("expected '/home/user/my-servers', got %q", result)
	}
}

func TestResolveStackFolderFlagOverridesEnv(t *testing.T) {
	t.Setenv("GOLOO_STACK_FOLDER", "/home/user/my-servers")
	command := &Command{FolderPath: "/opt/stacks"}
	result := resolveStackFolder(command)
	if result != "/opt/stacks" {
		t.Errorf("expected '/opt/stacks', got %q", result)
	}
}

func TestResolveStackFolderFlagWithoutEnv(t *testing.T) {
	t.Setenv("GOLOO_STACK_FOLDER", "")
	command := &Command{FolderPath: "/opt/stacks"}
	result := resolveStackFolder(command)
	if result != "/opt/stacks" {
		t.Errorf("expected '/opt/stacks', got %q", result)
	}
}
