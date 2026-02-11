package multipass

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/emergingrobotics/goloo/internal/config"
	"github.com/emergingrobotics/goloo/internal/provider"
)

type Provider struct {
	Verbose bool
}

func New(verbose bool) *Provider {
	return &Provider{Verbose: verbose}
}

func (p *Provider) Name() string {
	return "multipass"
}

func (p *Provider) Create(ctx context.Context, configuration *config.Config, cloudInitPath string) error {
	arguments := BuildLaunchArgs(configuration, cloudInitPath)

	if p.Verbose && cloudInitPath != "" {
		if err := p.launchWithCloudInitTailing(ctx, configuration.VM.Name, arguments); err != nil {
			return fmt.Errorf("multipass launch failed: %w", err)
		}
	} else {
		if err := p.runStreamingCommand(ctx, arguments...); err != nil {
			return fmt.Errorf("multipass launch failed: %w", err)
		}
	}

	configuration.Local = &config.LocalState{}

	p.verboseLog("getting VM info for %q", configuration.VM.Name)
	info, err := p.getInfo(ctx, configuration.VM.Name)
	if err != nil {
		return fmt.Errorf("failed to get VM info after creation: %w", err)
	}

	if len(info.IPv4) > 0 {
		configuration.Local.IP = info.IPv4[0]
		p.verboseLog("VM IP: %s", info.IPv4[0])
	}

	for _, mount := range configuration.VM.Mounts {
		mountArgs := []string{"mount", mount.Source, fmt.Sprintf("%s:%s", configuration.VM.Name, mount.Target)}
		if _, err := p.runCommand(ctx, mountArgs...); err != nil {
			return fmt.Errorf("failed to mount %s: %w", mount.Source, err)
		}
	}

	return nil
}

func (p *Provider) Delete(ctx context.Context, configuration *config.Config) error {
	if _, err := p.runCommand(ctx, "delete", configuration.VM.Name); err != nil {
		return fmt.Errorf("failed to delete VM %s: %w", configuration.VM.Name, err)
	}
	if _, err := p.runCommand(ctx, "purge"); err != nil {
		return fmt.Errorf("failed to purge deleted VMs: %w", err)
	}
	configuration.Local = nil
	return nil
}

func (p *Provider) Status(ctx context.Context, configuration *config.Config) (*provider.VMStatus, error) {
	info, err := p.getInfo(ctx, configuration.VM.Name)
	if err != nil {
		return nil, err
	}
	ip := ""
	if len(info.IPv4) > 0 {
		ip = info.IPv4[0]
	}
	return &provider.VMStatus{
		Name:     configuration.VM.Name,
		State:    info.State,
		IP:       ip,
		Provider: "multipass",
	}, nil
}

func (p *Provider) List(ctx context.Context) ([]provider.VMStatus, error) {
	output, err := p.runCommand(ctx, "list", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("multipass list failed: %w", err)
	}
	parsed, err := ParseListJSON(output)
	if err != nil {
		return nil, err
	}
	statuses := make([]provider.VMStatus, 0, len(parsed.List))
	for _, vm := range parsed.List {
		ip := ""
		if len(vm.IPv4) > 0 {
			ip = vm.IPv4[0]
		}
		statuses = append(statuses, provider.VMStatus{
			Name:     vm.Name,
			State:    vm.State,
			IP:       ip,
			Provider: "multipass",
		})
	}
	return statuses, nil
}

func (p *Provider) SSH(ctx context.Context, configuration *config.Config) error {
	command := exec.CommandContext(ctx, "multipass", "shell", configuration.VM.Name)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func (p *Provider) Stop(ctx context.Context, configuration *config.Config) error {
	if _, err := p.runCommand(ctx, "stop", configuration.VM.Name); err != nil {
		return fmt.Errorf("failed to stop VM %s: %w", configuration.VM.Name, err)
	}
	return nil
}

func (p *Provider) Start(ctx context.Context, configuration *config.Config) error {
	if _, err := p.runCommand(ctx, "start", configuration.VM.Name); err != nil {
		return fmt.Errorf("failed to start VM %s: %w", configuration.VM.Name, err)
	}
	return nil
}

func BuildLaunchArgs(configuration *config.Config, cloudInitPath string) []string {
	arguments := []string{"launch", configuration.VM.Image}
	arguments = append(arguments, "--name", configuration.VM.Name)

	if configuration.VM.CPUs > 0 {
		arguments = append(arguments, "--cpus", fmt.Sprintf("%d", configuration.VM.CPUs))
	}
	if configuration.VM.Memory != "" {
		arguments = append(arguments, "--memory", configuration.VM.Memory)
	}
	if configuration.VM.Disk != "" {
		arguments = append(arguments, "--disk", configuration.VM.Disk)
	}
	if cloudInitPath != "" {
		arguments = append(arguments, "--cloud-init", cloudInitPath)
	}

	return arguments
}

type MultipassInfo struct {
	Info map[string]MultipassVM `json:"info"`
}

type MultipassList struct {
	List []MultipassVM `json:"list"`
}

type MultipassVM struct {
	Name    string   `json:"name"`
	State   string   `json:"state"`
	IPv4    []string `json:"ipv4"`
	Release string   `json:"release"`
}

func ParseInfoJSON(data []byte) (*MultipassInfo, error) {
	var info MultipassInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse multipass info output: %w", err)
	}
	return &info, nil
}

func ParseListJSON(data []byte) (*MultipassList, error) {
	var list MultipassList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("failed to parse multipass list output: %w", err)
	}
	return &list, nil
}

func (p *Provider) getInfo(ctx context.Context, name string) (*MultipassVM, error) {
	output, err := p.runCommand(ctx, "info", name, "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("VM not found: check 'goloo list' for available VMs")
	}
	info, err := ParseInfoJSON(output)
	if err != nil {
		return nil, err
	}
	vm, exists := info.Info[name]
	if !exists {
		return nil, fmt.Errorf("VM %q not found in multipass info output", name)
	}
	return &vm, nil
}

func (p *Provider) verboseLog(format string, arguments ...interface{}) {
	if p.Verbose {
		fmt.Fprintf(os.Stderr, "[verbose] "+format+"\n", arguments...)
	}
}

func (p *Provider) runCommand(ctx context.Context, arguments ...string) ([]byte, error) {
	p.verboseLog("exec: multipass %s", strings.Join(arguments, " "))
	command := exec.CommandContext(ctx, "multipass", arguments...)
	return command.CombinedOutput()
}

func (p *Provider) runStreamingCommand(ctx context.Context, arguments ...string) error {
	p.verboseLog("exec: multipass %s", strings.Join(arguments, " "))
	command := exec.CommandContext(ctx, "multipass", arguments...)
	command.Stdout = os.Stderr
	command.Stderr = os.Stderr
	return command.Run()
}

func (p *Provider) launchWithCloudInitTailing(ctx context.Context, vmName string, arguments []string) error {
	p.verboseLog("exec: multipass %s", strings.Join(arguments, " "))
	launchCmd := exec.CommandContext(ctx, "multipass", arguments...)
	launchCmd.Stdout = os.Stderr
	launchCmd.Stderr = os.Stderr

	if err := launchCmd.Start(); err != nil {
		return err
	}

	tailCtx, tailCancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.tailCloudInitLog(tailCtx, vmName)
	}()

	launchErr := launchCmd.Wait()
	tailCancel()
	<-done

	return launchErr
}

func (p *Provider) tailCloudInitLog(ctx context.Context, vmName string) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}

		output, err := exec.CommandContext(ctx, "multipass", "info", vmName, "--format", "json").CombinedOutput()
		if err != nil {
			continue
		}
		info, err := ParseInfoJSON(output)
		if err != nil {
			continue
		}
		vm, exists := info.Info[vmName]
		if !exists || vm.State != "Running" {
			continue
		}
		break
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	p.verboseLog("streaming cloud-init log")
	tailCmd := exec.CommandContext(ctx, "multipass", "exec", vmName, "--",
		"tail", "-f", "/var/log/cloud-init-output.log")
	tailCmd.Stdout = os.Stderr
	tailCmd.Stderr = os.Stderr
	tailCmd.Run()
}
