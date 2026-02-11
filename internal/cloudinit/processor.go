package cloudinit

import (
	"fmt"
	"os"
	"strings"

	"github.com/emergingrobotics/goloo/internal/config"
)

type KeyFetchFunc func(username string) (string, error)

func Process(templatePath string, configuration *config.Config, fetchKeys KeyFetchFunc) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("cloud-init template not found: %s", templatePath)
	}

	users := getUsers(configuration)

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

	rendered := string(content)

	if strings.Contains(rendered, "{{") {
		templateData := buildTemplateData(configuration, keysPerUser)
		rendered, err = renderGoTemplate(rendered, templateData)
		if err != nil {
			return "", err
		}
	}

	rendered = substituteVariables(rendered, users, keysPerUser)

	temporaryFile, err := os.CreateTemp("", "goloo-cloudinit-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer temporaryFile.Close()

	if _, err := temporaryFile.WriteString(rendered); err != nil {
		os.Remove(temporaryFile.Name())
		return "", fmt.Errorf("failed to write processed cloud-init: %w", err)
	}

	return temporaryFile.Name(), nil
}

func getUsers(configuration *config.Config) []config.User {
	if configuration != nil && configuration.VM != nil {
		return configuration.VM.Users
	}
	return nil
}

func substituteVariables(content string, users []config.User, keysPerUser map[string]string) string {
	result := content

	if len(users) > 0 {
		firstUsername := users[0].Username
		if keys, exists := keysPerUser[firstUsername]; exists {
			result = substituteKey(result, "${SSH_PUBLIC_KEY}", keys)
		}
	}

	for username, keys := range keysPerUser {
		placeholder := fmt.Sprintf("${SSH_PUBLIC_KEY_%s}", strings.ToUpper(username))
		result = substituteKey(result, placeholder, keys)
	}

	return result
}

func substituteKey(content string, placeholder string, keys string) string {
	keyLines := splitKeys(keys)

	if len(keyLines) <= 1 {
		return strings.ReplaceAll(content, placeholder, keys)
	}

	contentLines := strings.Split(content, "\n")
	var result []string
	for _, line := range contentLines {
		if !strings.Contains(line, placeholder) {
			result = append(result, line)
			continue
		}

		index := strings.Index(line, placeholder)
		prefix := line[:index]

		for _, key := range keyLines {
			result = append(result, prefix+key)
		}
	}
	return strings.Join(result, "\n")
}

func splitKeys(keys string) []string {
	var result []string
	for _, line := range strings.Split(keys, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
