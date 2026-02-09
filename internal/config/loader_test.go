package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolvePathFromName(t *testing.T) {
	got := ResolvePath("devbox")
	want := filepath.Join("stacks", "devbox.json")
	if got != want {
		t.Errorf("ResolvePath(\"devbox\") = %q, want %q", got, want)
	}
}

func TestResolvePathFromJSONFilename(t *testing.T) {
	got := ResolvePath("devbox.json")
	want := "devbox.json"
	if got != want {
		t.Errorf("ResolvePath(\"devbox.json\") = %q, want %q", got, want)
	}
}

func TestResolvePathFromAbsolutePath(t *testing.T) {
	got := ResolvePath("/tmp/myconfig.json")
	want := "/tmp/myconfig.json"
	if got != want {
		t.Errorf("ResolvePath(\"/tmp/myconfig.json\") = %q, want %q", got, want)
	}
}

func TestResolvePathFromRelativeWithSeparator(t *testing.T) {
	got := ResolvePath("configs/devbox.json")
	want := "configs/devbox.json"
	if got != want {
		t.Errorf("ResolvePath(\"configs/devbox.json\") = %q, want %q", got, want)
	}
}

func TestLoadValidConfig(t *testing.T) {
	directory := t.TempDir()
	stacksDirectory := filepath.Join(directory, "stacks")
	os.MkdirAll(stacksDirectory, 0755)

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
	configPath := filepath.Join(directory, "stacks", "devbox.json")
	os.WriteFile(configPath, []byte(configJSON), 0644)

	configuration, path, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if path != configPath {
		t.Errorf("Load() path = %q, want %q", path, configPath)
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

func TestLoadConfigFileNotFound(t *testing.T) {
	_, _, err := Load("/nonexistent/path.json")
	if err == nil {
		t.Fatal("Load() should return error for nonexistent file")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(path, []byte("{not valid json}"), 0644)

	_, _, err := Load(path)
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
			Users: []User{{Username: "1baduser"}},
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
			Users: []User{{Username: "BadUser"}},
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
			Users: []User{{Username: "bad@user"}},
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
			Users: []User{{Username: "my-user_name"}},
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
			Users: []User{{Username: ""}},
		},
	}
	err := Validate(configuration)
	if err == nil {
		t.Fatal("Validate() should return error for empty username")
	}
}

func TestValidateNoUsers(t *testing.T) {
	configuration := &Config{
		VM: &VMConfig{
			Name: "devbox",
		},
	}
	if err := Validate(configuration); err != nil {
		t.Errorf("Validate() should allow config with no users: %v", err)
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

	path := filepath.Join(t.TempDir(), "roundtrip.json")
	if err := Save(path, original); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	loaded, loadedPath, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if loadedPath != path {
		t.Errorf("Load() path = %q, want %q", loadedPath, path)
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
			Name:     "devbox",
			PublicIP: "10.0.0.5",
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
	configJSON := `{"vm": {"name": "devbox"}}`
	path := filepath.Join(t.TempDir(), "minimal.json")
	os.WriteFile(path, []byte(configJSON), 0644)

	configuration, _, err := Load(path)
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
	configJSON := `{"vm": {"name": ""}}`
	path := filepath.Join(t.TempDir(), "invalid.json")
	os.WriteFile(path, []byte(configJSON), 0644)

	_, _, err := Load(path)
	if err == nil {
		t.Fatal("Load() should return validation error for empty name")
	}
}
