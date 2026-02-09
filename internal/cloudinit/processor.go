package cloudinit

import (
	"fmt"
	"os"
	"strings"

	"github.com/emergingrobotics/goloo/internal/config"
)

type KeyFetchFunc func(username string) (string, error)

func Process(templatePath string, users []config.User, fetchKeys KeyFetchFunc) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("cloud-init template not found: %s", templatePath)
	}

	keysPerUser := make(map[string]string)
	for _, user := range users {
		if user.GitHubUsername == "" {
			continue
		}
		keys, err := fetchKeys(user.GitHubUsername)
		if err != nil {
			return "", err
		}
		keysPerUser[user.Username] = keys
	}

	processed := substituteVariables(string(content), users, keysPerUser)

	temporaryFile, err := os.CreateTemp("", "goloo-cloudinit-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer temporaryFile.Close()

	if _, err := temporaryFile.WriteString(processed); err != nil {
		os.Remove(temporaryFile.Name())
		return "", fmt.Errorf("failed to write processed cloud-init: %w", err)
	}

	return temporaryFile.Name(), nil
}

func substituteVariables(content string, users []config.User, keysPerUser map[string]string) string {
	result := content

	if len(users) > 0 {
		firstUsername := users[0].Username
		if keys, exists := keysPerUser[firstUsername]; exists {
			result = strings.ReplaceAll(result, "${SSH_PUBLIC_KEY}", keys)
		}
	}

	for username, keys := range keysPerUser {
		placeholder := fmt.Sprintf("${SSH_PUBLIC_KEY_%s}", strings.ToUpper(username))
		result = strings.ReplaceAll(result, placeholder, keys)
	}

	return result
}
