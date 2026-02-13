package config

import (
	"encoding/json"
	"fmt"
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
