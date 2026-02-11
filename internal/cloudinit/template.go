package cloudinit

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/emergingrobotics/goloo/internal/config"
)

type TemplateUser struct {
	Username string
	SSHKeys  []string
}

type TemplateData struct {
	Name         string
	CPUs         int
	Memory       string
	Disk         string
	Image        string
	InstanceType string
	OS           string
	Region       string

	Hostname     string
	Domain       string
	FQDN         string
	TTL          int
	IsApexDomain bool
	CNAMEAliases []string

	Packages   []string
	WorkingDir string

	Users []TemplateUser

	Vars map[string]interface{}
}

func buildTemplateData(configuration *config.Config, keysPerUser map[string]string) TemplateData {
	data := TemplateData{
		Vars: make(map[string]interface{}),
	}

	if configuration.VM != nil {
		data.Name = configuration.VM.Name
		data.CPUs = configuration.VM.CPUs
		data.Memory = configuration.VM.Memory
		data.Disk = configuration.VM.Disk
		data.Image = configuration.VM.Image
		data.InstanceType = configuration.VM.InstanceType
		data.OS = configuration.VM.OS
		data.Region = configuration.VM.Region

		for _, user := range configuration.VM.Users {
			templateUser := TemplateUser{
				Username: user.Username,
			}
			if keys, exists := keysPerUser[user.Username]; exists {
				for _, key := range strings.Split(keys, "\n") {
					key = strings.TrimSpace(key)
					if key != "" {
						templateUser.SSHKeys = append(templateUser.SSHKeys, key)
					}
				}
			}
			data.Users = append(data.Users, templateUser)
		}
	}

	if configuration.DNS != nil {
		data.Hostname = configuration.DNS.Hostname
		data.Domain = configuration.DNS.Domain
		data.TTL = configuration.DNS.TTL
		data.IsApexDomain = configuration.DNS.IsApexDomain
		data.CNAMEAliases = configuration.DNS.CNAMEAliases
	}

	if data.Hostname == "" && configuration.VM != nil {
		data.Hostname = configuration.VM.Name
	}

	if data.Hostname != "" && data.Domain != "" {
		data.FQDN = data.Hostname + "." + data.Domain
	} else if data.Domain != "" {
		data.FQDN = data.Domain
	}

	if configuration.CloudInit != nil {
		data.Packages = configuration.CloudInit.Packages
		data.WorkingDir = configuration.CloudInit.WorkingDir
		if configuration.CloudInit.Vars != nil {
			data.Vars = configuration.CloudInit.Vars
		}
	}

	return data
}

func renderGoTemplate(content string, data TemplateData) (string, error) {
	tmpl, err := template.New("cloud-init").Option("missingkey=error").Parse(content)
	if err != nil {
		return "", fmt.Errorf("cloud-init template parse error: %w", err)
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("cloud-init template render error: %w", err)
	}

	return buffer.String(), nil
}
