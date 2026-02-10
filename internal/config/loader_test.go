package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveFolderJoinsComponents(t *testing.T) {
	got := ResolveFolder("stacks", "devbox")
	want := filepath.Join("stacks", "devbox")
	if got != want {
		t.Errorf("ResolveFolder(\"stacks\", \"devbox\") = %q, want %q", got, want)
	}
}

func TestConfigPathBuildsCorrectPath(t *testing.T) {
	got := ConfigPath("stacks", "devbox")
	want := filepath.Join("stacks", "devbox", "config.json")
	if got != want {
		t.Errorf("ConfigPath(\"stacks\", \"devbox\") = %q, want %q", got, want)
	}
}

func TestCloudInitPathBuildsCorrectPath(t *testing.T) {
	got := CloudInitPath("stacks", "devbox")
	want := filepath.Join("stacks", "devbox", "cloud-init.yaml")
	if got != want {
		t.Errorf("CloudInitPath(\"stacks\", \"devbox\") = %q, want %q", got, want)
	}
}

func TestConfigPathWithCustomFolder(t *testing.T) {
	got := ConfigPath("/home/user/servers", "web")
	want := filepath.Join("/home/user/servers", "web", "config.json")
	if got != want {
		t.Errorf("ConfigPath with custom folder = %q, want %q", got, want)
	}
}

func TestCloudInitPathWithCustomFolder(t *testing.T) {
	got := CloudInitPath("/home/user/servers", "web")
	want := filepath.Join("/home/user/servers", "web", "cloud-init.yaml")
	if got != want {
		t.Errorf("CloudInitPath with custom folder = %q, want %q", got, want)
	}
}

func TestLoadValidConfig(t *testing.T) {
	directory := t.TempDir()
	vmDirectory := filepath.Join(directory, "devbox")
	os.MkdirAll(vmDirectory, 0755)

	configJSON := `{
  "vm": {
    "name": "devbox",
    "cpus": 4,
    "memory": "4G",
    "disk": "40G",
    "image": "24.04",
    "users": [{"username": "ubuntu", "github_username": "gherlein"}]
  }
}`
	os.WriteFile(filepath.Join(vmDirectory, "config.json"), []byte(configJSON), 0644)

	configuration, path, err := Load(directory, "devbox")
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	expectedPath := filepath.Join(directory, "devbox", "config.json")
	if path != expectedPath {
		t.Errorf("Load() path = %q, want %q", path, expectedPath)
	}
	if configuration.VM.Name != "devbox" {
		t.Errorf("VM.Name = %q, want %q", configuration.VM.Name, "devbox")
	}
	if configuration.VM.CPUs != 4 {
		t.Errorf("VM.CPUs = %d, want %d", configuration.VM.CPUs, 4)
	}
	if configuration.VM.Memory != "4G" {
		t.Errorf("VM.Memory = %q, want %q", configuration.VM.Memory, "4G")
	}
}

func TestLoadConfigFolderNotFound(t *testing.T) {
	_, _, err := Load("/nonexistent", "devbox")
	if err == nil {
		t.Fatal("Load() should return error for nonexistent folder")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	directory := t.TempDir()
	vmDirectory := filepath.Join(directory, "bad")
	os.MkdirAll(vmDirectory, 0755)
	os.WriteFile(filepath.Join(vmDirectory, "config.json"), []byte("{not valid json}"), 0644)

	_, _, err := Load(directory, "bad")
	if err == nil {
		t.Fatal("Load() should return error for invalid JSON")
	}
}

func TestApplyDefaultsSetsValues(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{
			Name: "devbox",
		},
	}

	ApplyDefaults(configuration)

	if configuration.VM.CPUs != 2 {
		t.Errorf("Default CPUs = %d, want %d", configuration.VM.CPUs, 2)
	}
	if configuration.VM.Memory != "2G" {
		t.Errorf("Default Memory = %q, want %q", configuration.VM.Memory, "2G")
	}
	if configuration.VM.Disk != "20G" {
		t.Errorf("Default Disk = %q, want %q", configuration.VM.Disk, "20G")
	}
	if configuration.VM.Image != "24.04" {
		t.Errorf("Default Image = %q, want %q", configuration.VM.Image, "24.04")
	}
	if configuration.VM.Region != "us-east-1" {
		t.Errorf("Default Region = %q, want %q", configuration.VM.Region, "us-east-1")
	}
	if configuration.VM.InstanceType != "t3.micro" {
		t.Errorf("Default InstanceType = %q, want %q", configuration.VM.InstanceType, "t3.micro")
	}
}

func TestApplyDefaultsPreservesExistingValues(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{
			Name:   "devbox",
			CPUs:   8,
			Memory: "16G",
			Disk:   "100G",
			Image:  "22.04",
			Region: "eu-west-1",
		},
	}

	ApplyDefaults(configuration)

	if configuration.VM.CPUs != 8 {
		t.Errorf("CPUs should be preserved: got %d, want %d", configuration.VM.CPUs, 8)
	}
	if configuration.VM.Memory != "16G" {
		t.Errorf("Memory should be preserved: got %q, want %q", configuration.VM.Memory, "16G")
	}
	if configuration.VM.Disk != "100G" {
		t.Errorf("Disk should be preserved: got %q, want %q", configuration.VM.Disk, "100G")
	}
	if configuration.VM.Image != "22.04" {
		t.Errorf("Image should be preserved: got %q, want %q", configuration.VM.Image, "22.04")
	}
	if configuration.VM.Region != "eu-west-1" {
		t.Errorf("Region should be preserved: got %q, want %q", configuration.VM.Region, "eu-west-1")
	}
}

func TestApplyDefaultsDNSTTL(t *testing.T) {
	configuration := &Config{
		VM:  &VMConfig{Name: "devbox"},
		DNS: &DNSConfig{Domain: "example.com"},
	}

	ApplyDefaults(configuration)

	if configuration.DNS.TTL != 300 {
		t.Errorf("Default DNS TTL = %d, want %d", configuration.DNS.TTL, 300)
	}
}

func TestApplyDefaultsNilVM(t *testing.T) {
	configuration := &Config{}
	ApplyDefaults(configuration)
}

func TestValidateValid(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{
			Name: "devbox",
			Users: []User{
				{Username: "ubuntu", GitHubUsername: "gherlein"},
			},
		},
	}

	if err := Validate(configuration); err != nil {
		t.Errorf("Validate() returned error for valid config: %v", err)
	}
}

func TestValidateNilVM(t *testing.T) {
	configuration := &Config{}
	err := Validate(configuration)
	if err == nil {
		t.Fatal("Validate() should return error when vm section is nil")
	}
}

func TestValidateMissingName(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{},
	}
	err := Validate(configuration)
	if err == nil {
		t.Fatal("Validate() should return error when vm.name is empty")
	}
}

func TestValidateInvalidUsernameStartsWithNumber(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{
			Name:  "devbox",
			Users: []User{{Username: "1baduser", GitHubUsername: "test"}},
		},
	}
	err := Validate(configuration)
	if err == nil {
		t.Fatal("Validate() should return error for username starting with number")
	}
}

func TestValidateInvalidUsernameUppercase(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{
			Name:  "devbox",
			Users: []User{{Username: "BadUser", GitHubUsername: "test"}},
		},
	}
	err := Validate(configuration)
	if err == nil {
		t.Fatal("Validate() should return error for uppercase username")
	}
}

func TestValidateInvalidUsernameSpecialChars(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{
			Name:  "devbox",
			Users: []User{{Username: "bad@user", GitHubUsername: "test"}},
		},
	}
	err := Validate(configuration)
	if err == nil {
		t.Fatal("Validate() should return error for username with special characters")
	}
}

func TestValidateValidUsernameWithHyphensAndUnderscores(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{
			Name:  "devbox",
			Users: []User{{Username: "my-user_name", GitHubUsername: "test"}},
		},
	}
	if err := Validate(configuration); err != nil {
		t.Errorf("Validate() returned error for valid username with hyphens/underscores: %v", err)
	}
}

func TestValidateEmptyUsername(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{
			Name:  "devbox",
			Users: []User{{Username: "", GitHubUsername: "test"}},
		},
	}
	err := Validate(configuration)
	if err == nil {
		t.Fatal("Validate() should return error for empty username")
	}
}

func TestValidateMissingGitHubUsername(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{
			Name:  "devbox",
			Users: []User{{Username: "ubuntu"}},
		},
	}
	err := Validate(configuration)
	if err == nil {
		t.Fatal("Validate() should return error for missing github_username")
	}
}

func TestValidateDuplicateUsernames(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{
			Name: "devbox",
			Users: []User{
				{Username: "ubuntu", GitHubUsername: "alice"},
				{Username: "ubuntu", GitHubUsername: "bob"},
			},
		},
	}
	err := Validate(configuration)
	if err == nil {
		t.Fatal("Validate() should return error for duplicate usernames")
	}
}

func TestValidateCNAMEAliasesRequiresDomain(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{Name: "devbox"},
		DNS: &DNSConfig{
			CNAMEAliases: []string{"www"},
		},
	}
	err := Validate(configuration)
	if err == nil {
		t.Fatal("Validate() should return error for cname_aliases without domain")
	}
}

func TestValidateIsApexDomainRequiresDomain(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{Name: "devbox"},
		DNS: &DNSConfig{
			IsApexDomain: true,
		},
	}
	err := Validate(configuration)
	if err == nil {
		t.Fatal("Validate() should return error for is_apex_domain without domain")
	}
}

func TestValidateDNSWithDomainAndAliases(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{Name: "devbox", Users: []User{{Username: "ubuntu", GitHubUsername: "test"}}},
		DNS: &DNSConfig{
			Domain:       "example.com",
			CNAMEAliases: []string{"www"},
			IsApexDomain: true,
		},
	}
	if err := Validate(configuration); err != nil {
		t.Errorf("Validate() returned error for valid DNS config: %v", err)
	}
}

func TestValidateNoUsers(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{
			Name: "devbox",
		},
	}
	err := Validate(configuration)
	if err == nil {
		t.Fatal("Validate() should return error when no users defined")
	}
}

func TestSaveAndReload(t *testing.T) {
	original := &Config{
		VM: &VMConfig{
			Name:   "devbox",
			CPUs:   4,
			Memory: "4G",
			Disk:   "40G",
			Image:  "24.04",
			Users: []User{
				{Username: "ubuntu", GitHubUsername: "gherlein"},
			},
		},
		DNS: &DNSConfig{
			Hostname: "devbox",
			Domain:   "example.com",
			TTL:      300,
		},
	}

	directory := t.TempDir()
	vmDirectory := filepath.Join(directory, "devbox")
	os.MkdirAll(vmDirectory, 0755)
	configPath := filepath.Join(vmDirectory, "config.json")

	if err := Save(configPath, original); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	loaded, loadedPath, err := Load(directory, "devbox")
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if loadedPath != configPath {
		t.Errorf("Load() path = %q, want %q", loadedPath, configPath)
	}

	if !reflect.DeepEqual(original.VM.Name, loaded.VM.Name) {
		t.Errorf("VM.Name = %q, want %q", loaded.VM.Name, original.VM.Name)
	}
	if !reflect.DeepEqual(original.VM.Users, loaded.VM.Users) {
		t.Errorf("VM.Users = %+v, want %+v", loaded.VM.Users, original.VM.Users)
	}
	if !reflect.DeepEqual(original.DNS.Domain, loaded.DNS.Domain) {
		t.Errorf("DNS.Domain = %q, want %q", loaded.DNS.Domain, original.DNS.Domain)
	}
}

func TestSaveCreatesValidJSON(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{
			Name: "devbox",
		},
		Local: &LocalState{
			IP: "10.0.0.5",
		},
	}

	path := filepath.Join(t.TempDir(), "output.json")
	if err := Save(path, configuration); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	content := string(data)
	if content[len(content)-1] != '\n' {
		t.Error("Saved file should end with newline")
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	directory := t.TempDir()
	vmDirectory := filepath.Join(directory, "devbox")
	os.MkdirAll(vmDirectory, 0755)
	os.WriteFile(filepath.Join(vmDirectory, "config.json"), []byte(`{"vm": {"name": "devbox", "users": [{"username": "ubuntu", "github_username": "test"}]}}`), 0644)

	configuration, _, err := Load(directory, "devbox")
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if configuration.VM.CPUs != 2 {
		t.Errorf("Default CPUs not applied: got %d, want %d", configuration.VM.CPUs, 2)
	}
	if configuration.VM.Memory != "2G" {
		t.Errorf("Default Memory not applied: got %q, want %q", configuration.VM.Memory, "2G")
	}
}

func TestLoadValidatesAfterParsing(t *testing.T) {
	directory := t.TempDir()
	vmDirectory := filepath.Join(directory, "invalid")
	os.MkdirAll(vmDirectory, 0755)
	os.WriteFile(filepath.Join(vmDirectory, "config.json"), []byte(`{"vm": {"name": ""}}`), 0644)

	_, _, err := Load(directory, "invalid")
	if err == nil {
		t.Fatal("Load() should return validation error for empty name")
	}
}
