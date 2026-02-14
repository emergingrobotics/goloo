package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var validUsernamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

func ResolveFolder(folder string, name string) string {
	return filepath.Join(folder, name)
}

func ConfigPath(folder string, name string) string {
	return filepath.Join(ResolveFolder(folder, name), "config.json")
}

func CloudInitPath(folder string, name string) string {
	return filepath.Join(ResolveFolder(folder, name), "cloud-init.yaml")
}

func Load(folder string, name string) (*Config, string, error) {
	return LoadFromPath(ConfigPath(folder, name))
}

func LoadFromPath(path string) (*Config, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("config file not found: create %s", path)
	}

	var configuration Config
	if err := json.Unmarshal(data, &configuration); err != nil {
		return nil, "", fmt.Errorf("invalid JSON in %s: %w", path, err)
	}

	ApplyDefaults(&configuration)

	if err := Validate(&configuration); err != nil {
		return nil, "", err
	}

	return &configuration, path, nil
}

func Save(path string, configuration *Config) error {
	data, err := json.MarshalIndent(configuration, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func ApplyDefaults(configuration *Config) {
	if configuration.VM == nil {
		return
	}
	if configuration.VM.CPUs == 0 {
		configuration.VM.CPUs = 2
	}
	if configuration.VM.Memory == "" {
		configuration.VM.Memory = "2G"
	}
	if configuration.VM.Disk == "" {
		configuration.VM.Disk = "20G"
	}
	if configuration.VM.Image == "" {
		configuration.VM.Image = "24.04"
	}
	if configuration.VM.Region == "" {
		configuration.VM.Region = "us-east-1"
	}
	if configuration.VM.InstanceType == "" {
		configuration.VM.InstanceType = "t3.micro"
	}
	if configuration.DNS != nil && configuration.DNS.TTL == 0 {
		configuration.DNS.TTL = 300
	}
	if configuration.CloudInit == nil {
		configuration.CloudInit = &CloudInitConfig{}
	}
	if configuration.CloudInit.WorkingDir == "" {
		configuration.CloudInit.WorkingDir = "/var/www/html"
	}
}

func StatePath(folder, name, providerName string) string {
	return filepath.Join(ResolveFolder(folder, name), providerName, "config.json")
}

func StateCloudInitPath(folder, name, providerName string) string {
	return filepath.Join(ResolveFolder(folder, name), providerName, "cloud-init.yaml")
}

func SaveState(folder, name, providerName string, cfg *Config) error {
	stateDir := filepath.Join(ResolveFolder(folder, name), providerName)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory %s: %w", stateDir, err)
	}
	return Save(filepath.Join(stateDir, "config.json"), cfg)
}

func LoadState(folder, name, providerName string) (*Config, string, error) {
	return LoadFromPath(StatePath(folder, name, providerName))
}

func HasState(folder, name, providerName string) bool {
	_, err := os.Stat(StatePath(folder, name, providerName))
	return err == nil
}

func ClearState(folder, name, providerName string) error {
	stateDir := filepath.Join(ResolveFolder(folder, name), providerName)
	return os.RemoveAll(stateDir)
}

func CopyCloudInitToState(folder, name, providerName, cloudInitPath string) error {
	if cloudInitPath == "" {
		return nil
	}
	stateDir := filepath.Join(ResolveFolder(folder, name), providerName)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory %s: %w", stateDir, err)
	}

	src, err := os.Open(cloudInitPath)
	if err != nil {
		return fmt.Errorf("failed to open cloud-init file %s: %w", cloudInitPath, err)
	}
	defer src.Close()

	destPath := filepath.Join(stateDir, "cloud-init.yaml")
	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create state cloud-init file %s: %w", destPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy cloud-init to state: %w", err)
	}
	return nil
}

func Validate(configuration *Config) error {
	if configuration.VM == nil {
		return fmt.Errorf("config missing required 'vm' section")
	}
	if configuration.VM.Name == "" {
		return fmt.Errorf("config missing required field: vm.name")
	}
	if len(configuration.VM.Users) == 0 {
		return fmt.Errorf("config missing required field: vm.users (at least one user required)")
	}

	seen := make(map[string]bool)
	for _, user := range configuration.VM.Users {
		if user.Username == "" {
			return fmt.Errorf("user missing required field: username")
		}
		if !validUsernamePattern.MatchString(user.Username) {
			return fmt.Errorf("invalid username %q: must start with a lowercase letter and contain only lowercase letters, numbers, hyphens, underscores", user.Username)
		}
		if user.GitHubUsername == "" {
			return fmt.Errorf("user %q missing required field: github_username", user.Username)
		}
		if seen[user.Username] {
			return fmt.Errorf("duplicate username %q", user.Username)
		}
		seen[user.Username] = true
	}

	if configuration.DNS != nil {
		if len(configuration.DNS.CNAMEAliases) > 0 && configuration.DNS.Domain == "" {
			return fmt.Errorf("dns.cname_aliases requires dns.domain")
		}
		if configuration.DNS.IsApexDomain && configuration.DNS.Domain == "" {
			return fmt.Errorf("dns.is_apex_domain requires dns.domain")
		}
	}

	return nil
}
