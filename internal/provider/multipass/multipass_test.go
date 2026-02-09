package multipass

import (
	"reflect"
	"testing"

	"github.com/emergingrobotics/goloo/internal/config"
)

func TestProviderName(t *testing.T) {
	provider := New()
	if provider.Name() != "multipass" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "multipass")
	}
}

func TestBuildLaunchArgsFullConfig(t *testing.T) {
	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:   "devbox",
			Image:  "24.04",
			CPUs:   4,
			Memory: "4G",
			Disk:   "40G",
		},
	}

	arguments := BuildLaunchArgs(configuration, "/tmp/cloud-init.yaml")

	expected := []string{
		"launch", "24.04",
		"--name", "devbox",
		"--cpus", "4",
		"--memory", "4G",
		"--disk", "40G",
		"--cloud-init", "/tmp/cloud-init.yaml",
	}

	if !reflect.DeepEqual(arguments, expected) {
		t.Errorf("BuildLaunchArgs() = %v, want %v", arguments, expected)
	}
}

func TestBuildLaunchArgsMinimal(t *testing.T) {
	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:  "minimal",
			Image: "24.04",
		},
	}

	arguments := BuildLaunchArgs(configuration, "")

	expected := []string{
		"launch", "24.04",
		"--name", "minimal",
	}

	if !reflect.DeepEqual(arguments, expected) {
		t.Errorf("BuildLaunchArgs() = %v, want %v", arguments, expected)
	}
}

func TestBuildLaunchArgsWithCloudInit(t *testing.T) {
	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:  "withci",
			Image: "22.04",
		},
	}

	arguments := BuildLaunchArgs(configuration, "/path/to/cloud-init.yaml")

	foundCloudInit := false
	for i, arg := range arguments {
		if arg == "--cloud-init" && i+1 < len(arguments) {
			if arguments[i+1] != "/path/to/cloud-init.yaml" {
				t.Errorf("--cloud-init value = %q, want %q", arguments[i+1], "/path/to/cloud-init.yaml")
			}
			foundCloudInit = true
			break
		}
	}
	if !foundCloudInit {
		t.Error("BuildLaunchArgs() should include --cloud-init flag when path is provided")
	}
}

func TestBuildLaunchArgsNoCloudInitWhenEmpty(t *testing.T) {
	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:  "noci",
			Image: "24.04",
		},
	}

	arguments := BuildLaunchArgs(configuration, "")

	for _, arg := range arguments {
		if arg == "--cloud-init" {
			t.Error("BuildLaunchArgs() should not include --cloud-init when path is empty")
		}
	}
}

func TestParseInfoJSON(t *testing.T) {
	jsonData := []byte(`{
		"info": {
			"devbox": {
				"name": "devbox",
				"state": "Running",
				"ipv4": ["192.168.64.5"],
				"release": "Ubuntu 24.04 LTS"
			}
		}
	}`)

	info, err := ParseInfoJSON(jsonData)
	if err != nil {
		t.Fatalf("ParseInfoJSON() returned error: %v", err)
	}

	vm, exists := info.Info["devbox"]
	if !exists {
		t.Fatal("ParseInfoJSON() should contain 'devbox' entry")
	}
	if vm.Name != "devbox" {
		t.Errorf("VM.Name = %q, want %q", vm.Name, "devbox")
	}
	if vm.State != "Running" {
		t.Errorf("VM.State = %q, want %q", vm.State, "Running")
	}
	if len(vm.IPv4) != 1 || vm.IPv4[0] != "192.168.64.5" {
		t.Errorf("VM.IPv4 = %v, want [\"192.168.64.5\"]", vm.IPv4)
	}
	if vm.Release != "Ubuntu 24.04 LTS" {
		t.Errorf("VM.Release = %q, want %q", vm.Release, "Ubuntu 24.04 LTS")
	}
}

func TestParseInfoJSONMultipleVMs(t *testing.T) {
	jsonData := []byte(`{
		"info": {
			"vm1": {"name": "vm1", "state": "Running", "ipv4": ["10.0.0.1"], "release": "Ubuntu 24.04 LTS"},
			"vm2": {"name": "vm2", "state": "Stopped", "ipv4": [], "release": "Ubuntu 22.04 LTS"}
		}
	}`)

	info, err := ParseInfoJSON(jsonData)
	if err != nil {
		t.Fatalf("ParseInfoJSON() returned error: %v", err)
	}

	if len(info.Info) != 2 {
		t.Errorf("ParseInfoJSON() returned %d VMs, want 2", len(info.Info))
	}

	vm2 := info.Info["vm2"]
	if vm2.State != "Stopped" {
		t.Errorf("vm2.State = %q, want %q", vm2.State, "Stopped")
	}
	if len(vm2.IPv4) != 0 {
		t.Errorf("vm2.IPv4 = %v, want empty", vm2.IPv4)
	}
}

func TestParseInfoJSONInvalid(t *testing.T) {
	_, err := ParseInfoJSON([]byte("not json"))
	if err == nil {
		t.Fatal("ParseInfoJSON() should return error for invalid JSON")
	}
}

func TestParseListJSON(t *testing.T) {
	jsonData := []byte(`{
		"list": [
			{"name": "devbox", "state": "Running", "ipv4": ["192.168.64.5"], "release": "Ubuntu 24.04 LTS"},
			{"name": "testvm", "state": "Stopped", "ipv4": [], "release": "Ubuntu 22.04 LTS"}
		]
	}`)

	list, err := ParseListJSON(jsonData)
	if err != nil {
		t.Fatalf("ParseListJSON() returned error: %v", err)
	}

	if len(list.List) != 2 {
		t.Fatalf("ParseListJSON() returned %d VMs, want 2", len(list.List))
	}

	if list.List[0].Name != "devbox" {
		t.Errorf("List[0].Name = %q, want %q", list.List[0].Name, "devbox")
	}
	if list.List[0].State != "Running" {
		t.Errorf("List[0].State = %q, want %q", list.List[0].State, "Running")
	}
	if list.List[1].Name != "testvm" {
		t.Errorf("List[1].Name = %q, want %q", list.List[1].Name, "testvm")
	}
	if list.List[1].State != "Stopped" {
		t.Errorf("List[1].State = %q, want %q", list.List[1].State, "Stopped")
	}
}

func TestParseListJSONEmpty(t *testing.T) {
	jsonData := []byte(`{"list": []}`)

	list, err := ParseListJSON(jsonData)
	if err != nil {
		t.Fatalf("ParseListJSON() returned error: %v", err)
	}

	if len(list.List) != 0 {
		t.Errorf("ParseListJSON() returned %d VMs for empty list, want 0", len(list.List))
	}
}

func TestParseListJSONInvalid(t *testing.T) {
	_, err := ParseListJSON([]byte("{bad json"))
	if err == nil {
		t.Fatal("ParseListJSON() should return error for invalid JSON")
	}
}

func TestBuildLaunchArgsStartsWithLaunchAndImage(t *testing.T) {
	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:   "vm1",
			Image:  "22.04",
			CPUs:   2,
			Memory: "2G",
		},
	}

	arguments := BuildLaunchArgs(configuration, "")

	if len(arguments) < 2 {
		t.Fatalf("BuildLaunchArgs() returned %d args, want at least 2", len(arguments))
	}
	if arguments[0] != "launch" {
		t.Errorf("First arg = %q, want %q", arguments[0], "launch")
	}
	if arguments[1] != "22.04" {
		t.Errorf("Second arg (image) = %q, want %q", arguments[1], "22.04")
	}
}
