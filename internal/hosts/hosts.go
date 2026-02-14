package hosts

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const hostsFile = "/etc/hosts"

var hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$`)

func ValidateIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %q", ip)
	}
	return nil
}

func ValidateHostname(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}
	if len(hostname) > 253 {
		return fmt.Errorf("hostname too long: %d characters (max 253)", len(hostname))
	}
	if !hostnameRegex.MatchString(hostname) {
		return fmt.Errorf("invalid hostname %q: must contain only alphanumeric characters, hyphens, and dots", hostname)
	}
	return nil
}

func BuildHostnames(vmName, dnsHostname, dnsDomain string) []string {
	name := dnsHostname
	if name == "" {
		name = vmName
	}

	var hostnames []string
	if dnsDomain != "" {
		hostnames = append(hostnames, name+"."+dnsDomain)
	}
	hostnames = append(hostnames, name)
	return hostnames
}

func startMarker(vmName string) string {
	return "# goloo:" + vmName
}

func endMarker(vmName string) string {
	return "# /goloo:" + vmName
}

func buildBlock(vmName, ip string, hostnames []string) string {
	return startMarker(vmName) + "\n" +
		ip + "    " + strings.Join(hostnames, " ") + "\n" +
		endMarker(vmName) + "\n"
}

func removeBlock(content, vmName string) string {
	start := startMarker(vmName)
	end := endMarker(vmName)

	lines := strings.Split(content, "\n")
	var result []string
	inside := false
	for _, line := range lines {
		if strings.TrimSpace(line) == start {
			inside = true
			continue
		}
		if inside && strings.TrimSpace(line) == end {
			inside = false
			continue
		}
		if !inside {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func Add(vmName, ip string, hostnames []string, verbose bool) error {
	if err := ValidateIP(ip); err != nil {
		return err
	}
	for _, h := range hostnames {
		if err := ValidateHostname(h); err != nil {
			return err
		}
	}

	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", hostsFile, err)
	}

	cleaned := removeBlock(string(content), vmName)
	if !strings.HasSuffix(cleaned, "\n") {
		cleaned += "\n"
	}
	updated := cleaned + buildBlock(vmName, ip, hostnames)

	cmd := exec.Command("sudo", "tee", hostsFile)
	cmd.Stdin = strings.NewReader(updated)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write %s (sudo required): %w", hostsFile, err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[verbose] added /etc/hosts entry: %s -> %s\n", ip, strings.Join(hostnames, " "))
	}
	return nil
}

func Remove(vmName string, verbose bool) error {
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", hostsFile, err)
	}

	cleaned := removeBlock(string(content), vmName)
	if string(content) == cleaned {
		return nil
	}

	cmd := exec.Command("sudo", "tee", hostsFile)
	cmd.Stdin = strings.NewReader(cleaned)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write %s (sudo required): %w", hostsFile, err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[verbose] removed /etc/hosts entry for %s\n", vmName)
	}
	return nil
}

func HasEntry(vmName string) bool {
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), startMarker(vmName))
}

func ManualInstructions(ip string, hostnames []string, vmName string) string {
	return fmt.Sprintf("Add this line to /etc/hosts manually:\n  %s    %s",
		ip, strings.Join(hostnames, " "))
}
