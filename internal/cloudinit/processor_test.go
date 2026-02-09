package cloudinit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emergingrobotics/goloo/internal/config"
)

func TestFetchGitHubKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/testuser.keys" {
			writer.WriteHeader(http.StatusOK)
			fmt.Fprint(writer, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAItest testuser@github")
			return
		}
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	originalBaseURL := GitHubBaseURL
	GitHubBaseURL = server.URL
	defer func() { GitHubBaseURL = originalBaseURL }()

	keys, err := FetchGitHubKeys("testuser")
	if err != nil {
		t.Fatalf("FetchGitHubKeys() returned error: %v", err)
	}

	expected := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAItest testuser@github"
	if keys != expected {
		t.Errorf("FetchGitHubKeys() = %q, want %q", keys, expected)
	}
}

func TestFetchGitHubKeysUserNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	originalBaseURL := GitHubBaseURL
	GitHubBaseURL = server.URL
	defer func() { GitHubBaseURL = originalBaseURL }()

	_, err := FetchGitHubKeys("nonexistent")
	if err == nil {
		t.Fatal("FetchGitHubKeys() should return error for 404 response")
	}
	if !strings.Contains(err.Error(), "no SSH keys found") {
		t.Errorf("Error should mention 'no SSH keys found', got: %v", err)
	}
}

func TestFetchGitHubKeysServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalBaseURL := GitHubBaseURL
	GitHubBaseURL = server.URL
	defer func() { GitHubBaseURL = originalBaseURL }()

	_, err := FetchGitHubKeys("testuser")
	if err == nil {
		t.Fatal("FetchGitHubKeys() should return error for 500 response")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("Error should mention HTTP status code, got: %v", err)
	}
}

func TestFetchGitHubKeysEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		fmt.Fprint(writer, "")
	}))
	defer server.Close()

	originalBaseURL := GitHubBaseURL
	GitHubBaseURL = server.URL
	defer func() { GitHubBaseURL = originalBaseURL }()

	_, err := FetchGitHubKeys("testuser")
	if err == nil {
		t.Fatal("FetchGitHubKeys() should return error for empty keys response")
	}
}

func TestFetchGitHubKeysMultipleKeys(t *testing.T) {
	keysResponse := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIkey1\nssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABkey2"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		fmt.Fprint(writer, keysResponse)
	}))
	defer server.Close()

	originalBaseURL := GitHubBaseURL
	GitHubBaseURL = server.URL
	defer func() { GitHubBaseURL = originalBaseURL }()

	keys, err := FetchGitHubKeys("testuser")
	if err != nil {
		t.Fatalf("FetchGitHubKeys() returned error: %v", err)
	}
	if keys != keysResponse {
		t.Errorf("FetchGitHubKeys() = %q, want %q", keys, keysResponse)
	}
}

func TestSubstituteDefaultVariable(t *testing.T) {
	content := "#cloud-config\nusers:\n  - name: ubuntu\n    ssh_authorized_keys:\n      - ${SSH_PUBLIC_KEY}"
	users := []config.User{{Username: "ubuntu", GitHubUsername: "gherlein"}}
	keysPerUser := map[string]string{"ubuntu": "ssh-ed25519 AAAAC3 testkey"}

	result := substituteVariables(content, users, keysPerUser)

	if !strings.Contains(result, "ssh-ed25519 AAAAC3 testkey") {
		t.Errorf("Result should contain substituted key, got: %s", result)
	}
	if strings.Contains(result, "${SSH_PUBLIC_KEY}") {
		t.Error("Result should not contain unsubstituted ${SSH_PUBLIC_KEY} placeholder")
	}
}

func TestSubstitutePerUserVariable(t *testing.T) {
	content := "ssh_authorized_keys:\n  - ${SSH_PUBLIC_KEY_UBUNTU}\n  - ${SSH_PUBLIC_KEY_ADMIN}"
	users := []config.User{
		{Username: "ubuntu", GitHubUsername: "user1"},
		{Username: "admin", GitHubUsername: "user2"},
	}
	keysPerUser := map[string]string{
		"ubuntu": "ssh-ed25519 key-ubuntu",
		"admin":  "ssh-ed25519 key-admin",
	}

	result := substituteVariables(content, users, keysPerUser)

	if !strings.Contains(result, "ssh-ed25519 key-ubuntu") {
		t.Error("Result should contain ubuntu's key")
	}
	if !strings.Contains(result, "ssh-ed25519 key-admin") {
		t.Error("Result should contain admin's key")
	}
	if strings.Contains(result, "${SSH_PUBLIC_KEY_UBUNTU}") {
		t.Error("Result should not contain unsubstituted ${SSH_PUBLIC_KEY_UBUNTU}")
	}
	if strings.Contains(result, "${SSH_PUBLIC_KEY_ADMIN}") {
		t.Error("Result should not contain unsubstituted ${SSH_PUBLIC_KEY_ADMIN}")
	}
}

func TestSubstituteNoUsers(t *testing.T) {
	content := "some content with ${SSH_PUBLIC_KEY}"
	result := substituteVariables(content, nil, map[string]string{})

	if result != content {
		t.Errorf("With no users, content should be unchanged: got %q, want %q", result, content)
	}
}

func TestProcessValidTemplate(t *testing.T) {
	templateContent := "#cloud-config\nusers:\n  - name: ubuntu\n    ssh_authorized_keys:\n      - ${SSH_PUBLIC_KEY}"
	templatePath := filepath.Join(t.TempDir(), "test.yaml")
	os.WriteFile(templatePath, []byte(templateContent), 0644)

	users := []config.User{{Username: "ubuntu", GitHubUsername: "testuser"}}

	fakeKeyFetcher := func(username string) (string, error) {
		return "ssh-ed25519 AAAAC3 fakekey", nil
	}

	resultPath, err := Process(templatePath, users, fakeKeyFetcher)
	if err != nil {
		t.Fatalf("Process() returned error: %v", err)
	}
	defer os.Remove(resultPath)

	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("Failed to read result file: %v", err)
	}

	result := string(data)
	if !strings.Contains(result, "ssh-ed25519 AAAAC3 fakekey") {
		t.Error("Processed template should contain the fetched SSH key")
	}
	if strings.Contains(result, "${SSH_PUBLIC_KEY}") {
		t.Error("Processed template should not contain unsubstituted placeholder")
	}
}

func TestProcessMissingTemplate(t *testing.T) {
	_, err := Process("/nonexistent/template.yaml", nil, FetchGitHubKeys)
	if err == nil {
		t.Fatal("Process() should return error for missing template file")
	}
}

func TestProcessNoUsers(t *testing.T) {
	templateContent := "#cloud-config\npackages:\n  - git"
	templatePath := filepath.Join(t.TempDir(), "nouser.yaml")
	os.WriteFile(templatePath, []byte(templateContent), 0644)

	resultPath, err := Process(templatePath, nil, FetchGitHubKeys)
	if err != nil {
		t.Fatalf("Process() returned error: %v", err)
	}
	defer os.Remove(resultPath)

	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("Failed to read result file: %v", err)
	}

	if string(data) != templateContent {
		t.Errorf("With no users, template should pass through unchanged: got %q, want %q", string(data), templateContent)
	}
}

func TestProcessKeyFetchError(t *testing.T) {
	templatePath := filepath.Join(t.TempDir(), "template.yaml")
	os.WriteFile(templatePath, []byte("#cloud-config"), 0644)

	users := []config.User{{Username: "ubuntu", GitHubUsername: "baduser"}}

	failingFetcher := func(username string) (string, error) {
		return "", fmt.Errorf("network error")
	}

	_, err := Process(templatePath, users, failingFetcher)
	if err == nil {
		t.Fatal("Process() should propagate key fetch errors")
	}
}

func TestProcessSkipsUsersWithoutGitHub(t *testing.T) {
	templateContent := "#cloud-config\nkeys: ${SSH_PUBLIC_KEY}"
	templatePath := filepath.Join(t.TempDir(), "template.yaml")
	os.WriteFile(templatePath, []byte(templateContent), 0644)

	users := []config.User{
		{Username: "ubuntu", GitHubUsername: ""},
		{Username: "admin", GitHubUsername: "validuser"},
	}

	fetcher := func(username string) (string, error) {
		if username == "validuser" {
			return "ssh-ed25519 valid-key", nil
		}
		return "", fmt.Errorf("should not fetch for empty github username")
	}

	resultPath, err := Process(templatePath, users, fetcher)
	if err != nil {
		t.Fatalf("Process() returned error: %v", err)
	}
	defer os.Remove(resultPath)
}
