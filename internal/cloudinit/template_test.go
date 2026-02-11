package cloudinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emergingrobotics/goloo/internal/config"
)

func TestBuildTemplateDataFromFullConfig(t *testing.T) {
	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:         "devbox",
			CPUs:         4,
			Memory:       "4G",
			Disk:         "40G",
			Image:        "24.04",
			InstanceType: "t3.small",
			OS:           "ubuntu-24.04",
			Region:       "us-east-1",
			Users: []config.User{
				{Username: "ubuntu", GitHubUsername: "testuser"},
			},
		},
		DNS: &config.DNSConfig{
			Hostname:     "web",
			Domain:       "example.com",
			TTL:          300,
			IsApexDomain: true,
			CNAMEAliases: []string{"www"},
		},
		CloudInit: &config.CloudInitConfig{
			Packages:   []string{"nginx", "git"},
			WorkingDir: "/opt/app",
			Vars:       map[string]interface{}{"app_name": "myapp", "port": 8080},
		},
	}
	keysPerUser := map[string]string{"ubuntu": "ssh-ed25519 AAAA key1"}

	data := buildTemplateData(configuration, keysPerUser)

	if data.Name != "devbox" {
		t.Errorf("Name = %q, want %q", data.Name, "devbox")
	}
	if data.CPUs != 4 {
		t.Errorf("CPUs = %d, want 4", data.CPUs)
	}
	if data.Memory != "4G" {
		t.Errorf("Memory = %q, want %q", data.Memory, "4G")
	}
	if data.Disk != "40G" {
		t.Errorf("Disk = %q, want %q", data.Disk, "40G")
	}
	if data.Image != "24.04" {
		t.Errorf("Image = %q, want %q", data.Image, "24.04")
	}
	if data.InstanceType != "t3.small" {
		t.Errorf("InstanceType = %q, want %q", data.InstanceType, "t3.small")
	}
	if data.OS != "ubuntu-24.04" {
		t.Errorf("OS = %q, want %q", data.OS, "ubuntu-24.04")
	}
	if data.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", data.Region, "us-east-1")
	}
	if data.Hostname != "web" {
		t.Errorf("Hostname = %q, want %q", data.Hostname, "web")
	}
	if data.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", data.Domain, "example.com")
	}
	if data.FQDN != "web.example.com" {
		t.Errorf("FQDN = %q, want %q", data.FQDN, "web.example.com")
	}
	if data.TTL != 300 {
		t.Errorf("TTL = %d, want 300", data.TTL)
	}
	if !data.IsApexDomain {
		t.Error("IsApexDomain should be true")
	}
	if len(data.CNAMEAliases) != 1 || data.CNAMEAliases[0] != "www" {
		t.Errorf("CNAMEAliases = %v, want [www]", data.CNAMEAliases)
	}
	if len(data.Packages) != 2 {
		t.Errorf("Packages length = %d, want 2", len(data.Packages))
	}
	if data.WorkingDir != "/opt/app" {
		t.Errorf("WorkingDir = %q, want %q", data.WorkingDir, "/opt/app")
	}
	if data.Vars["app_name"] != "myapp" {
		t.Errorf("Vars[app_name] = %v, want %q", data.Vars["app_name"], "myapp")
	}
	if len(data.Users) != 1 {
		t.Fatalf("Users length = %d, want 1", len(data.Users))
	}
	if data.Users[0].Username != "ubuntu" {
		t.Errorf("Users[0].Username = %q, want %q", data.Users[0].Username, "ubuntu")
	}
	if len(data.Users[0].SSHKeys) != 1 || data.Users[0].SSHKeys[0] != "ssh-ed25519 AAAA key1" {
		t.Errorf("Users[0].SSHKeys = %v, want [ssh-ed25519 AAAA key1]", data.Users[0].SSHKeys)
	}
}

func TestBuildTemplateDataNilSections(t *testing.T) {
	data := buildTemplateData(&config.Config{}, map[string]string{})

	if data.Name != "" {
		t.Errorf("Name should be empty, got %q", data.Name)
	}
	if data.Hostname != "" {
		t.Errorf("Hostname should be empty, got %q", data.Hostname)
	}
	if data.Vars == nil {
		t.Error("Vars should be initialized to empty map, not nil")
	}
}

func TestBuildTemplateDataHostnameFallsBackToVMName(t *testing.T) {
	configuration := &config.Config{
		VM: &config.VMConfig{Name: "devbox"},
	}
	data := buildTemplateData(configuration, map[string]string{})

	if data.Hostname != "devbox" {
		t.Errorf("Hostname should fall back to vm.name, got %q", data.Hostname)
	}
}

func TestBuildTemplateDataHostnameFallsBackWithDomain(t *testing.T) {
	configuration := &config.Config{
		VM:  &config.VMConfig{Name: "devbox"},
		DNS: &config.DNSConfig{Domain: "example.com"},
	}
	data := buildTemplateData(configuration, map[string]string{})

	if data.Hostname != "devbox" {
		t.Errorf("Hostname should fall back to vm.name, got %q", data.Hostname)
	}
	if data.FQDN != "devbox.example.com" {
		t.Errorf("FQDN = %q, want %q", data.FQDN, "devbox.example.com")
	}
}

func TestBuildTemplateDataDNSHostnameOverridesVMName(t *testing.T) {
	configuration := &config.Config{
		VM:  &config.VMConfig{Name: "devbox"},
		DNS: &config.DNSConfig{Hostname: "web", Domain: "example.com"},
	}
	data := buildTemplateData(configuration, map[string]string{})

	if data.Hostname != "web" {
		t.Errorf("Hostname should use dns.hostname, got %q", data.Hostname)
	}
	if data.FQDN != "web.example.com" {
		t.Errorf("FQDN = %q, want %q", data.FQDN, "web.example.com")
	}
}

func TestBuildTemplateDataMultipleSSHKeys(t *testing.T) {
	configuration := &config.Config{
		VM: &config.VMConfig{
			Users: []config.User{{Username: "ubuntu", GitHubUsername: "testuser"}},
		},
	}
	keysPerUser := map[string]string{
		"ubuntu": "ssh-ed25519 key1\nssh-rsa key2\nssh-ed25519 key3",
	}
	data := buildTemplateData(configuration, keysPerUser)

	if len(data.Users[0].SSHKeys) != 3 {
		t.Errorf("SSHKeys length = %d, want 3", len(data.Users[0].SSHKeys))
	}
}

func TestRenderGoTemplateSimple(t *testing.T) {
	content := "hostname: {{.Hostname}}\ndomain: {{.Domain}}"
	data := TemplateData{Hostname: "web", Domain: "example.com"}

	result, err := renderGoTemplate(content, data)
	if err != nil {
		t.Fatalf("renderGoTemplate() error: %v", err)
	}

	expected := "hostname: web\ndomain: example.com"
	if result != expected {
		t.Errorf("renderGoTemplate() = %q, want %q", result, expected)
	}
}

func TestRenderGoTemplateWithRange(t *testing.T) {
	content := "packages:\n{{- range .Packages}}\n  - {{.}}\n{{- end}}"
	data := TemplateData{Packages: []string{"nginx", "git", "htop"}}

	result, err := renderGoTemplate(content, data)
	if err != nil {
		t.Fatalf("renderGoTemplate() error: %v", err)
	}

	if !strings.Contains(result, "- nginx") {
		t.Error("Result should contain nginx")
	}
	if !strings.Contains(result, "- git") {
		t.Error("Result should contain git")
	}
	if !strings.Contains(result, "- htop") {
		t.Error("Result should contain htop")
	}
}

func TestRenderGoTemplateWithConditional(t *testing.T) {
	content := "{{if .IsApexDomain}}apex: true{{else}}apex: false{{end}}"

	dataTrue := TemplateData{IsApexDomain: true}
	result, err := renderGoTemplate(content, dataTrue)
	if err != nil {
		t.Fatalf("renderGoTemplate() error: %v", err)
	}
	if result != "apex: true" {
		t.Errorf("result = %q, want %q", result, "apex: true")
	}

	dataFalse := TemplateData{IsApexDomain: false}
	result, err = renderGoTemplate(content, dataFalse)
	if err != nil {
		t.Fatalf("renderGoTemplate() error: %v", err)
	}
	if result != "apex: false" {
		t.Errorf("result = %q, want %q", result, "apex: false")
	}
}

func TestRenderGoTemplateWithVars(t *testing.T) {
	content := "app: {{index .Vars \"app_name\"}}"
	data := TemplateData{
		Vars: map[string]interface{}{"app_name": "myservice"},
	}

	result, err := renderGoTemplate(content, data)
	if err != nil {
		t.Fatalf("renderGoTemplate() error: %v", err)
	}
	if result != "app: myservice" {
		t.Errorf("result = %q, want %q", result, "app: myservice")
	}
}

func TestRenderGoTemplateWithUsers(t *testing.T) {
	content := "{{range .Users}}user: {{.Username}}\n{{range .SSHKeys}}  key: {{.}}\n{{end}}{{end}}"
	data := TemplateData{
		Users: []TemplateUser{
			{Username: "ubuntu", SSHKeys: []string{"ssh-ed25519 key1", "ssh-rsa key2"}},
		},
	}

	result, err := renderGoTemplate(content, data)
	if err != nil {
		t.Fatalf("renderGoTemplate() error: %v", err)
	}

	if !strings.Contains(result, "user: ubuntu") {
		t.Error("Result should contain 'user: ubuntu'")
	}
	if !strings.Contains(result, "key: ssh-ed25519 key1") {
		t.Error("Result should contain first SSH key")
	}
	if !strings.Contains(result, "key: ssh-rsa key2") {
		t.Error("Result should contain second SSH key")
	}
}

func TestRenderGoTemplateMissingKeyReturnsError(t *testing.T) {
	content := "host: {{.Hostnam}}"
	data := TemplateData{Hostname: "web"}

	_, err := renderGoTemplate(content, data)
	if err == nil {
		t.Fatal("renderGoTemplate() should return error for unknown field .Hostnam")
	}
}

func TestRenderGoTemplateInvalidSyntaxReturnsError(t *testing.T) {
	content := "host: {{.Hostname"
	data := TemplateData{}

	_, err := renderGoTemplate(content, data)
	if err == nil {
		t.Fatal("renderGoTemplate() should return error for unclosed template delimiter")
	}
}

func TestRenderGoTemplatePreservesLegacyPlaceholders(t *testing.T) {
	content := "keys:\n  - ${SSH_PUBLIC_KEY}\nhostname: {{.Hostname}}"
	data := TemplateData{Hostname: "web"}

	result, err := renderGoTemplate(content, data)
	if err != nil {
		t.Fatalf("renderGoTemplate() error: %v", err)
	}

	if !strings.Contains(result, "${SSH_PUBLIC_KEY}") {
		t.Error("Legacy placeholder should be preserved by Go template rendering")
	}
	if !strings.Contains(result, "hostname: web") {
		t.Error("Go template should have rendered hostname")
	}
}

func TestRenderGoTemplateNoDelimitersPassthrough(t *testing.T) {
	content := "#cloud-config\npackages:\n  - git\n  - nginx"
	data := TemplateData{}

	result, err := renderGoTemplate(content, data)
	if err != nil {
		t.Fatalf("renderGoTemplate() error: %v", err)
	}
	if result != content {
		t.Errorf("Content without template delimiters should pass through unchanged")
	}
}

func TestProcessGoTemplateAndLegacyTogether(t *testing.T) {
	templateContent := "#cloud-config\nhostname: {{.Hostname}}\npackages:\n{{- range .Packages}}\n  - {{.}}\n{{- end}}\nusers:\n  - name: ubuntu\n    ssh_authorized_keys:\n      - ${SSH_PUBLIC_KEY}"

	templatePath := filepath.Join(t.TempDir(), "combined.yaml")
	os.WriteFile(templatePath, []byte(templateContent), 0644)

	configuration := &config.Config{
		VM: &config.VMConfig{
			Users: []config.User{{Username: "ubuntu", GitHubUsername: "testuser"}},
		},
		DNS: &config.DNSConfig{Hostname: "web"},
		CloudInit: &config.CloudInitConfig{
			Packages: []string{"nginx", "git"},
		},
	}

	fetcher := func(username string) (string, error) {
		return "ssh-ed25519 AAAA fakekey", nil
	}

	resultPath, err := Process(templatePath, configuration, fetcher)
	if err != nil {
		t.Fatalf("Process() returned error: %v", err)
	}
	defer os.Remove(resultPath)

	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("Failed to read result file: %v", err)
	}

	result := string(data)
	if !strings.Contains(result, "hostname: web") {
		t.Error("Go template should have rendered hostname")
	}
	if !strings.Contains(result, "- nginx") {
		t.Error("Go template should have rendered packages")
	}
	if !strings.Contains(result, "- git") {
		t.Error("Go template should have rendered packages")
	}
	if !strings.Contains(result, "ssh-ed25519 AAAA fakekey") {
		t.Error("Legacy SSH key substitution should have run")
	}
	if strings.Contains(result, "${SSH_PUBLIC_KEY}") {
		t.Error("Legacy placeholder should have been substituted")
	}
	if strings.Contains(result, "{{") {
		t.Error("Go template delimiters should not remain in output")
	}
}

func TestProcessWithCNAMEAliasesTemplate(t *testing.T) {
	templateContent := "{{.FQDN}}\n{{- range .CNAMEAliases}}\n{{.}}.{{$.Domain}}\n{{- end}}"

	templatePath := filepath.Join(t.TempDir(), "cname.yaml")
	os.WriteFile(templatePath, []byte(templateContent), 0644)

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name: "web",
		},
		DNS: &config.DNSConfig{
			Hostname:     "web",
			Domain:       "example.com",
			CNAMEAliases: []string{"www", "api"},
		},
	}

	fetcher := func(username string) (string, error) {
		return "", nil
	}

	resultPath, err := Process(templatePath, configuration, fetcher)
	if err != nil {
		t.Fatalf("Process() returned error: %v", err)
	}
	defer os.Remove(resultPath)

	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("Failed to read result file: %v", err)
	}

	result := string(data)
	if !strings.Contains(result, "web.example.com") {
		t.Error("Result should contain FQDN")
	}
	if !strings.Contains(result, "www.example.com") {
		t.Error("Result should contain www CNAME alias")
	}
	if !strings.Contains(result, "api.example.com") {
		t.Error("Result should contain api CNAME alias")
	}
}
