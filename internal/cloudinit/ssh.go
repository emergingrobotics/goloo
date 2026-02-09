package cloudinit

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

var GitHubBaseURL = "https://github.com"

func FetchGitHubKeys(username string) (string, error) {
	url := fmt.Sprintf("%s/%s.keys", GitHubBaseURL, username)
	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch SSH keys for %s: %w", username, err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("no SSH keys found for GitHub user %q: verify username at github.com/%s.keys", username, username)
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch SSH keys for %s: HTTP %d", username, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read SSH keys for %s: %w", username, err)
	}

	keys := strings.TrimSpace(string(body))
	if keys == "" {
		return "", fmt.Errorf("no SSH keys found for GitHub user %q: add keys at github.com/settings/keys", username)
	}

	return keys, nil
}
