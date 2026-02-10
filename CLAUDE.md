# Goloo Project Instructions

## What This Project Is

Goloo is a CLI tool that provisions virtual machines locally (via Multipass) or in AWS (via AWS SDK) using identical configuration files. The goal is infrastructure-as-code that's transparent to where it runs.

## Core Principle

**One config, any target.** A goloo config file plus a cloud-init YAML should:
1. Create a local VM via `goloo create myvm`
2. Create an identical AWS EC2 instance via `goloo create myvm --aws`

The developer workflow is: develop locally, deploy to cloud with a flag change.

## How It Works

### Local VMs (Default)

Goloo shells out to `multipass`. The config file translates directly to multipass CLI parameters:

```
goloo config          →    multipass command
─────────────────────────────────────────────
vm.name               →    --name
vm.cpus               →    --cpus
vm.memory             →    --memory
vm.disk               →    --disk
vm.image              →    (positional arg)
cloud-init.yaml       →    --cloud-init
```

The cloud-init YAML passes through unchanged. Multipass and AWS EC2 both use cloud-init natively.

### AWS VMs (--aws flag)

Goloo uses the AWS Go SDK to:
1. Create a CloudFormation stack (EC2 instance + security group)
2. Pass the cloud-init YAML as EC2 UserData
3. Create Route53 DNS record: `{hostname}.{domain}` → instance IP

DNS uses:
- `dns.hostname` from config (or `vm.name` if not specified)
- `dns.domain` from config (optional; DNS records won't be created if absent)

## Config Layout

Each VM's configuration lives in a named folder containing `config.json` and an optional `cloud-init.yaml`:

```
stacks/devbox/
├── config.json
└── cloud-init.yaml
```

`config.json`:

```json
{
  "vm": {
    "name": "devbox",
    "cpus": 4,
    "memory": "4G",
    "disk": "40G",
    "image": "24.04",
    "users": [
      {"username": "ubuntu", "github_username": "gherlein"}
    ]
  },
  "dns": {
    "hostname": "devbox",
    "domain": "example.com"
  }
}
```

For local VMs, `dns` section is ignored. For AWS, it's optional — DNS records are only created when the section is present. Cloud-init is optional — if `cloud-init.yaml` doesn't exist in the folder, the VM is created without it.

## Cloud-Init Files

Standard cloud-init YAML with one extension - SSH key placeholder:

```yaml
#cloud-config
users:
  - name: ubuntu
    ssh_authorized_keys:
      - ${SSH_PUBLIC_KEY}
```

Goloo fetches SSH keys from GitHub (`https://github.com/{username}.keys`) and substitutes before passing to multipass/AWS.

## Project Structure

```
goloo/
├── cmd/goloo/          # CLI entry point
├── internal/
│   ├── config/         # Config loading, validation
│   ├── cloudinit/      # Cloud-init processing, SSH key fetching
│   └── provider/
│       ├── interface.go
│       ├── multipass/  # Shells to multipass CLI
│       └── aws/        # AWS SDK (CloudFormation, Route53)
├── stacks/             # VM config folders (gitignored)
│   └── <name>/
│       ├── config.json
│       └── cloud-init.yaml
├── configs/            # Shared cloud-init templates
└── docs/
    ├── DESIGN.md       # Architecture and diagrams
    ├── PLAN.md         # Development plan
    └── goloo-notes.md  # Detailed design notes
```

## Key Design Decisions

### Shell Out to Multipass (Not gRPC)

Multipass has an internal gRPC API but it's undocumented and requires TLS certificates. The CLI is the stable public interface. We shell out and parse JSON output (`--format json`).

### AWS via SDK (Not CLI)

Unlike multipass, AWS has excellent Go SDK support. We use:
- `aws-sdk-go-v2/service/cloudformation` - Stack management
- `aws-sdk-go-v2/service/ec2` - VPC/Subnet discovery
- `aws-sdk-go-v2/service/route53` - DNS records
- `aws-sdk-go-v2/service/ssm` - AMI ID lookup

### Config Files are Multipass-Compatible

A goloo config should be usable with raw multipass:

```bash
# These should produce equivalent VMs:
goloo create devbox

multipass launch 24.04 \
  --name devbox \
  --cpus 4 \
  --memory 4G \
  --disk 40G \
  --cloud-init stacks/devbox/cloud-init.yaml
```

## CLI Commands

```bash
goloo create <name>                    # Local VM from stacks/<name>/
goloo create <name> --aws              # AWS EC2 instance
goloo create <name> -f ~/my-servers    # Use ~/my-servers/<name>/
goloo delete <name>                    # Delete (detects provider from config)
goloo list                             # List all VMs (both providers)
goloo ssh <name>                       # SSH into VM
goloo status <name>                    # Show VM status
goloo stop <name>                      # Stop a VM
goloo start <name>                     # Start a stopped VM
```

## Testing Requirements

- Unit tests for config parsing, cloud-init processing
- Integration tests require actual multipass/AWS (manual or CI)
- All phases must pass tests before proceeding to next phase
- Never skip tests

## Future Scope

Reserved for future implementation (not in current scope):
- GCP Compute Engine provider
- Azure VM provider
- DigitalOcean Droplets provider

Current scope is **Multipass + AWS only**.

## Code Style

- Go standard formatting (`gofmt`)
- No comments that describe what code does (code should be self-documenting)
- Comments only for why, not what
- Error messages should be actionable
- No abbreviations in names (`instance` not `inst`)

## Build Commands

```bash
make build      # Build binary to ./bin/goloo
make run-tests  # Run all tests
make clean      # Remove build artifacts
make install    # Install to ~/bin
```

## Dependencies

- Go 1.21+
- Multipass (for local VMs)
- AWS credentials configured (for --aws)

## References

- [docs/DESIGN.md](docs/DESIGN.md) - Architecture, diagrams, data flow
- [docs/PLAN.md](docs/PLAN.md) - Development phases and acceptance criteria
- [docs/goloo-notes.md](docs/goloo-notes.md) - Detailed implementation notes
- [docs/about-the-name.md](docs/about-the-name.md) - Why "goloo"
