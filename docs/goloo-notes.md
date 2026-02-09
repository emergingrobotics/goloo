# Goloo: Unified VM Provisioning Tool

A Go CLI tool that provisions VMs locally (Multipass) or in the cloud (AWS) using the same cloud-init configurations.

## Name Origin

**goloo** = Go (language/gopher) + Leeloo (The Fifth Element character who says "multipass")

Short, memorable, no trademark issues, fun to say.

## The Problem

From research notes:

> What's missing is a tool that literally does:
> ```bash
> mytool launch --cloud-init config.yaml          # runs locally
> mytool launch --cloud-init config.yaml --on aws  # runs in AWS
> ```
>
> This feels like an obvious gap. The pieces are all there (cloud-init is universal, cloud APIs exist), but nobody has built a polished CLI that unifies local and cloud VM lifecycle with a single config.

**Cloud-init is already portable** - the same YAML works in Multipass, AWS EC2, GCP, Azure, DigitalOcean. The missing piece is a unified CLI.

## Base: aws-ec2 Tool

The [emergingrobotics/aws-ec2](https://github.com/emergingrobotics/aws-ec2) tool provides an excellent foundation:

### Current Features
- Go CLI with `-c` (create) and `-d` (delete) flags
- JSON config files in `stacks/` directory
- GitHub SSH key fetching
- User creation with passwordless sudo
- Route53 DNS integration
- CloudFormation for reliable provisioning
- Nested config structure (`vm`, `dns` sections)
- Multiple OS support via SSM parameter lookup

### Architecture
```
Config (JSON)
    ├── vm: VMConfig (instance_type, os, users, cloud-init)
    └── dns: DNSConfig (hostname, domain, target_ip)
                ↓
           AWS Backend
    ├── CloudFormation (EC2 provisioning)
    ├── SSM (AMI lookup)
    └── Route53 (DNS)
```

## Proposed Goloo Architecture

### Provider Pattern

```
Config (JSON)
    ├── provider: "multipass" | "aws" | "gcp" | ...
    ├── vm: VMConfig
    └── dns: DNSConfig (optional)
                ↓
        Provider Dispatcher
           /          \
    Multipass       AWS Backend
    Provider        (existing)
```

### Directory Structure

```
goloo/
├── main.go              # CLI entry point, provider dispatcher
├── config.go            # Unified config handling
├── providers/
│   ├── interface.go     # Provider interface definition
│   ├── multipass/
│   │   └── multipass.go # Multipass implementation
│   └── aws/
│       └── aws.go       # Existing AWS logic (refactored)
├── cloudinit/
│   └── transform.go     # SSH key injection, template processing
├── stacks/              # Config files (gitignored)
└── cloud-init/          # Cloud-init templates
```

### Provider Interface

```go
type VMProvider interface {
    Create(ctx context.Context, cfg *VMConfig) error
    Delete(ctx context.Context, cfg *VMConfig) error
    Status(ctx context.Context, cfg *VMConfig) (*VMStatus, error)
    SSH(ctx context.Context, cfg *VMConfig) error
    List(ctx context.Context) ([]VMStatus, error)
}

type VMStatus struct {
    Name      string
    State     string    // running, stopped, etc.
    IP        string
    Provider  string
    CreatedAt time.Time
}
```

### Unified Config Structure

```go
type Config struct {
    Provider string     `json:"provider"` // "multipass", "aws", "gcp"
    VM       *VMConfig  `json:"vm,omitempty"`
    DNS      *DNSConfig `json:"dns,omitempty"`
}

type VMConfig struct {
    // Common fields (all providers)
    Name          string   `json:"name"`
    CPUs          int      `json:"cpus"`
    Memory        string   `json:"memory"`           // "4G"
    Disk          string   `json:"disk"`             // "40G"
    Image         string   `json:"image"`            // "24.04" or "ubuntu-24.04"
    CloudInitFile string   `json:"cloud_init_file"`
    Users         []User   `json:"users"`
    Mounts        []Mount  `json:"mounts,omitempty"` // Host mounts

    // AWS-specific (ignored for multipass)
    InstanceType  string   `json:"instance_type,omitempty"`
    VpcID         string   `json:"vpc_id,omitempty"`
    SubnetID      string   `json:"subnet_id,omitempty"`
    Region        string   `json:"region,omitempty"`

    // Output fields (provider fills in)
    PublicIP      string   `json:"public_ip,omitempty"`
    InstanceID    string   `json:"instance_id,omitempty"`
    State         string   `json:"state,omitempty"`
}

type User struct {
    Username       string `json:"username"`
    GitHubUsername string `json:"github_username"`
}

type Mount struct {
    Source string `json:"source"` // Host path
    Target string `json:"target"` // VM path
}
```

## Config Examples

### Local Development (Multipass)

```json
{
  "provider": "multipass",
  "vm": {
    "name": "dev",
    "cpus": 4,
    "memory": "4G",
    "disk": "40G",
    "image": "24.04",
    "cloud_init_file": "cloud-init/dev.yaml",
    "users": [
      {"username": "admin", "github_username": "gherlein"}
    ],
    "mounts": [
      {"source": "/Users/gherlein/src", "target": "/home/admin/src"}
    ]
  }
}
```

### Cloud Development (AWS)

```json
{
  "provider": "aws",
  "vm": {
    "name": "dev",
    "instance_type": "t3.medium",
    "image": "ubuntu-24.04",
    "region": "us-west-2",
    "cloud_init_file": "cloud-init/dev.yaml",
    "users": [
      {"username": "admin", "github_username": "gherlein"}
    ]
  },
  "dns": {
    "hostname": "dev",
    "domain": "example.com",
    "ttl": 300
  }
}
```

### Full Claude Development Environment

```json
{
  "provider": "multipass",
  "vm": {
    "name": "claude-dev",
    "cpus": 4,
    "memory": "8G",
    "disk": "80G",
    "image": "24.04",
    "cloud_init_file": "cloud-init/claude-dev.yaml",
    "users": [
      {"username": "ubuntu", "github_username": "gherlein"}
    ]
  }
}
```

## CLI Design

### Commands

```bash
# Create VM
goloo create <name>                    # Uses provider from config
goloo create <name> --local            # Force Multipass
goloo create <name> --aws              # Force AWS
goloo -c -n <name>                     # Short form (aws-ec2 compatible)

# Delete VM
goloo delete <name>
goloo -d -n <name>                     # Short form

# Status/Info
goloo status <name>
goloo info <name>

# List VMs
goloo list                             # All providers
goloo list --local                     # Multipass only
goloo list --aws                       # AWS only

# SSH Access
goloo ssh <name>
goloo shell <name>                     # Alias

# VM Lifecycle
goloo stop <name>
goloo start <name>
goloo restart <name>

# Mounts (Multipass only)
goloo mount <name> /host/path:/vm/path
goloo unmount <name> /vm/path

# Snapshots
goloo snapshot <name> <snapshot-name>
goloo restore <name> <snapshot-name>
```

### Flag Compatibility

Maintain backwards compatibility with aws-ec2:

```bash
goloo -c -n mystack              # Create (aws-ec2 style)
goloo -d -n mystack              # Delete (aws-ec2 style)
goloo create mystack             # Create (new style)
goloo delete mystack             # Delete (new style)
```

## Multipass Provider Implementation

```go
package multipass

import (
    "context"
    "fmt"
    "os/exec"
    "strconv"
)

type Provider struct{}

func (p *Provider) Create(ctx context.Context, cfg *VMConfig) error {
    // 1. Process cloud-init (substitute SSH keys)
    cloudInitPath, err := processCloudInit(cfg.CloudInitFile, cfg.Users)
    if err != nil {
        return fmt.Errorf("cloud-init processing failed: %w", err)
    }
    defer os.Remove(cloudInitPath) // Clean up temp file

    // 2. Build multipass command
    args := []string{
        "launch", cfg.Image,
        "--name", cfg.Name,
        "--cpus", strconv.Itoa(cfg.CPUs),
        "--memory", cfg.Memory,
        "--disk", cfg.Disk,
        "--cloud-init", cloudInitPath,
    }

    // 3. Execute
    cmd := exec.CommandContext(ctx, "multipass", args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Run(); err != nil {
        return fmt.Errorf("multipass launch failed: %w", err)
    }

    // 4. Setup mounts
    for _, mount := range cfg.Mounts {
        mountArgs := []string{"mount", mount.Source, fmt.Sprintf("%s:%s", cfg.Name, mount.Target)}
        if err := exec.CommandContext(ctx, "multipass", mountArgs...).Run(); err != nil {
            return fmt.Errorf("mount failed: %w", err)
        }
    }

    // 5. Get IP and update config
    ip, err := p.getIP(ctx, cfg.Name)
    if err == nil {
        cfg.PublicIP = ip
    }
    cfg.State = "running"

    return nil
}

func (p *Provider) Delete(ctx context.Context, cfg *VMConfig) error {
    // Delete and purge
    cmd := exec.CommandContext(ctx, "multipass", "delete", cfg.Name)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("delete failed: %w", err)
    }

    cmd = exec.CommandContext(ctx, "multipass", "purge")
    return cmd.Run()
}

func (p *Provider) SSH(ctx context.Context, cfg *VMConfig) error {
    cmd := exec.CommandContext(ctx, "multipass", "shell", cfg.Name)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func (p *Provider) getIP(ctx context.Context, name string) (string, error) {
    cmd := exec.CommandContext(ctx, "multipass", "info", name, "--format", "csv")
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    // Parse CSV to extract IP
    // ...
    return ip, nil
}
```

## Cloud-Init Processing

Shared logic for both providers:

```go
package cloudinit

import (
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "strings"
)

func ProcessCloudInit(templatePath string, users []User) (string, error) {
    // Read template
    content, err := ioutil.ReadFile(templatePath)
    if err != nil {
        return "", err
    }

    // Fetch SSH keys for each user
    for _, user := range users {
        keys, err := fetchGitHubKeys(user.GitHubUsername)
        if err != nil {
            return "", fmt.Errorf("failed to fetch keys for %s: %w", user.GitHubUsername, err)
        }

        // Replace placeholder
        placeholder := fmt.Sprintf("${SSH_PUBLIC_KEY_%s}", strings.ToUpper(user.Username))
        content = bytes.ReplaceAll(content, []byte(placeholder), []byte(keys))

        // Also replace generic placeholder
        content = bytes.ReplaceAll(content, []byte("${SSH_PUBLIC_KEY}"), []byte(keys))
    }

    // Write to temp file
    tmpFile, err := ioutil.TempFile("", "cloud-init-*.yaml")
    if err != nil {
        return "", err
    }

    if _, err := tmpFile.Write(content); err != nil {
        return "", err
    }

    return tmpFile.Name(), tmpFile.Close()
}

func fetchGitHubKeys(username string) (string, error) {
    url := fmt.Sprintf("https://github.com/%s.keys", username)
    resp, err := http.Get(url)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    keys, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    return string(keys), nil
}
```

## Implementation Phases

### Phase 1: Refactor aws-ec2
- Extract provider interface
- Move AWS logic into `providers/aws/`
- Keep backwards compatibility
- No new features, just restructure

### Phase 2: Add Multipass Provider
- Implement `providers/multipass/`
- Add `--local` flag
- Shared cloud-init processing
- Test with existing cloud-init configs

### Phase 3: Unified CLI
- Add new command style (`goloo create` vs `goloo -c`)
- Add `list`, `ssh`, `status` commands
- Provider auto-detection from config

### Phase 4: Enhanced Features
- Snapshot support
- Mount management
- Config validation
- Shell completions

### Phase 5: Additional Providers (Future)
- GCP
- Azure
- DigitalOcean
- Hetzner

## Shared Components

Code that works for both providers:

| Component | Description |
|-----------|-------------|
| GitHub SSH key fetching | `https://github.com/<user>.keys` |
| Cloud-init template processing | Variable substitution |
| Config file management | JSON read/write |
| Random hostname generation | For avoiding DNS conflicts |
| User creation logic | Embedded in cloud-init |

## Testing Strategy

```bash
# Unit tests
go test ./...

# Integration tests (require multipass/AWS)
go test -tags=integration ./...

# Manual testing matrix
goloo create test-local --local          # Multipass
goloo create test-aws --aws              # AWS
goloo ssh test-local
goloo ssh test-aws
goloo delete test-local
goloo delete test-aws
```

## Migration from aws-ec2

Existing aws-ec2 configs work unchanged:

```bash
# Old way (still works)
ec2 -c -n mystack

# New way (same config)
goloo -c -n mystack
goloo create mystack
```

The tool detects legacy flat config format and converts internally to nested format.

## References

- [Ubuntu Multipass](https://multipass.run/)
- [cloud-init Documentation](https://cloudinit.readthedocs.io/)
- [aws-ec2 Repository](https://github.com/emergingrobotics/aws-ec2)
- [The Fifth Element](https://en.wikipedia.org/wiki/The_Fifth_Element) - Origin of "multipass" and "Leeloo"
