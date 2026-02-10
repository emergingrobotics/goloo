# Goloo

<img src="./images/goloo-logo.png" alt="Goloo" width="50%">

Provision identical servers locally or in AWS from a single config file.

## The Problem

You want to build a server. You need it running locally for development, then deploy the same thing to AWS for production. Normally this means maintaining two separate setups — Vagrant/Multipass configs for local, Terraform/CloudFormation for cloud — that inevitably drift apart.

Goloo uses one JSON config and one cloud-init YAML to create the same server in both places.

## About the Name

**Go** + **Leeloo**. Named after the character from The Fifth Element who carries a "multipass" — the same Multipass that Canonical named their VM manager after. Goloo extends that idea: one tool, one config, anywhere you need to go. See [docs/about-the-name.md](docs/about-the-name.md) for the full story.

## Quick Start

### 1. Create a config folder

Each VM config lives in its own folder under `stacks/`. The folder contains a `config.json` and an optional `cloud-init.yaml`.

Create `stacks/web-server/config.json`:

```json
{
  "vm": {
    "name": "web-server",
    "cpus": 2,
    "memory": "2G",
    "disk": "20G",
    "image": "24.04",
    "users": [
      {"username": "ubuntu", "github_username": "your-github-username"}
    ]
  }
}
```

### 2. Write a cloud-init file

This is a standard [cloud-init](https://cloudinit.readthedocs.io/) YAML that defines what your server looks like. Goloo fetches SSH keys from GitHub automatically using the `${SSH_PUBLIC_KEY}` placeholder.

Create `stacks/web-server/cloud-init.yaml`:

```yaml
#cloud-config
users:
  - name: ubuntu
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    ssh_authorized_keys:
      - ${SSH_PUBLIC_KEY}

package_update: true
package_upgrade: true

packages:
  - nginx
  - certbot
  - python3-certbot-nginx
  - fail2ban
  - ufw

runcmd:
  - ufw allow OpenSSH
  - ufw allow 'Nginx Full'
  - ufw --force enable
  - systemctl enable nginx
  - systemctl start nginx
```

### 3. Create the server locally

```bash
goloo create web-server
```

This launches a Multipass VM with nginx, certbot, and fail2ban installed. SSH in with:

```bash
goloo ssh web-server
```

### 4. Deploy the same server to AWS

See `examples/aws-web-server/` for a complete AWS config with all parameters (`instance_type`, `os`, `region`, `dns`). Copy it and edit:

```bash
mkdir -p stacks/aws-web-server
cp examples/aws-web-server/config.json examples/aws-web-server/cloud-init.yaml stacks/aws-web-server/
# edit stacks/aws-web-server/config.json with your domain and GitHub username

goloo create aws-web-server --aws
```

Or skip editing the config entirely with the `-u` flag:

```bash
goloo create aws-web-server --aws -f ./examples/ -u gherlein
```

Goloo creates a CloudFormation stack with an EC2 instance, security group, and (if dns is configured) a Route53 A record pointing `web.example.com` at the instance IP.

## Example: Building a Development Server

A more complete example — a polyglot dev server with Docker, Go, Node.js, and Python:

`stacks/dev/config.json`:

```json
{
  "vm": {
    "name": "dev",
    "cpus": 4,
    "memory": "4G",
    "disk": "40G",
    "image": "24.04",
    "instance_type": "t3.medium",
    "users": [
      {"username": "ubuntu", "github_username": "your-github-username"}
    ]
  }
}
```

Several cloud-init templates are included in `configs/` for common setups. Copy one to `stacks/dev/cloud-init.yaml`:

| Config | What it installs |
|--------|-----------------|
| `configs/base.yaml` | curl, wget, vim, htop, git |
| `configs/dev.yaml` | build-essential, ripgrep, tmux, fzf, jq |
| `configs/python-dev.yaml` | Python 3, pip, venv, uv |
| `configs/go-dev.yaml` | Go 1.23, gopls, delve debugger |
| `configs/node-dev.yaml` | Node.js 22, npm, pnpm, yarn |
| `configs/claude-dev.yaml` | All of the above plus Docker |

Develop locally:

```bash
goloo create dev
goloo ssh dev
```

When it's ready for the cloud:

```bash
goloo create dev --aws
```

## Commands

```
goloo create <name>             Create a local VM (Multipass)
goloo create <name> --aws       Create an AWS EC2 instance
goloo delete <name>             Delete VM (auto-detects provider)
goloo list                      List VMs
goloo list --aws                List AWS VMs
goloo ssh <name>                SSH into VM
goloo status <name>             Show VM status
goloo stop <name>               Stop VM
goloo start <name>              Start VM
goloo dns swap <name>           Update DNS A record to current VM IP
```

### Flags

| Flag | Description |
|------|-------------|
| `--aws` | Use AWS provider |
| `--local` | Use local Multipass provider |
| `--folder`, `-f PATH` | Base folder for configs (default: `stacks/`) |
| `--users`, `-u USERS` | GitHub usernames for SSH key injection (comma-separated) |
| `--verbose`, `-v` | Show detailed progress |
| `--version` | Show version |
| `--help`, `-h` | Show help |

Flags can go in any order:

```bash
goloo create web-server --aws
goloo create --aws web-server -f ~/my-servers
goloo create web-server -u gherlein
goloo create web-server -u "alice,bob"
```

### The `--users` flag

The `--users`/`-u` flag provides GitHub usernames whose SSH public keys are fetched and injected into the cloud-init template. This overrides any users defined in the config JSON.

```bash
goloo create web-server -f ./examples/ -u gherlein
```

The first username maps to the `ubuntu` VM user (the default for Multipass and most cloud-init templates). Additional usernames become both the VM user and the GitHub lookup:

```bash
goloo create web-server -u "alice,deploy-bot"
# alice  → VM user "ubuntu", SSH keys from github.com/alice.keys
# deploy-bot → VM user "deploy-bot", SSH keys from github.com/deploy-bot.keys
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GOLOO_STACK_FOLDER` | Default base folder for configs (overridden by `--folder`/`-f`) |

Precedence: `--folder`/`-f` flag > `GOLOO_STACK_FOLDER` > `stacks/`

### Provider Auto-Detection

When you don't pass `--aws` or `--local`, goloo detects the provider from the config:

1. Config has `stack_id` (previously created with AWS) → AWS
2. Otherwise → Multipass

A config can include a `dns` section without triggering AWS — DNS records are only created when you explicitly pass `--aws`. This means `goloo delete web-server` does the right thing regardless of where the VM was created.

### Legacy Flags

For users migrating from [aws-ec2](https://github.com/emergingrobotics/aws-ec2):

```bash
goloo -c -n web-server    # Same as: goloo create web-server --aws
goloo -d -n web-server    # Same as: goloo delete web-server --aws
```

## Config Reference

### vm section

| Field | Default | Description |
|-------|---------|-------------|
| `name` | (required) | VM name, used for stack naming and config lookup |
| `cpus` | 2 | Number of CPUs (Multipass) |
| `memory` | "2G" | RAM allocation (Multipass) |
| `disk` | "20G" | Disk size (Multipass) |
| `image` | "24.04" | Ubuntu version (Multipass) |
| `instance_type` | "t3.micro" | EC2 instance type (AWS) |
| `os` | "ubuntu-24.04" | AMI lookup key (AWS) |
| `region` | "us-east-1" | AWS region |
| `users` | | List of `{"username", "github_username"}` for SSH key injection |
| `vpc_id` | | Specific VPC (AWS, auto-discovered if omitted) |
| `subnet_id` | | Specific subnet (AWS, auto-discovered if omitted) |
| `mounts` | | List of `{"source", "target"}` host mounts (Multipass) |

### dns section (optional, AWS only)

| Field | Default | Description |
|-------|---------|-------------|
| `hostname` | vm.name | Hostname portion of the DNS record |
| `domain` | | Route53 hosted zone domain |
| `ttl` | 300 | DNS record TTL in seconds |
| `zone_id` | | Route53 zone ID (auto-looked up from domain if omitted) |
| `is_apex_domain` | false | Create an A record at the zone apex (bare domain) |
| `cname_aliases` | | Additional CNAME records pointing at the hostname (e.g., `["www"]`) |

### Supported AWS operating systems

`ubuntu-24.04`, `ubuntu-22.04`, `ubuntu-20.04`, `amazon-linux-2023`, `amazon-linux-2`, `debian-12`, `debian-11`

See `examples/aws-web-server/` for a complete config with all AWS fields populated.

## Cloud-Init Variables

Goloo substitutes these placeholders in cloud-init YAML before passing to the provider:

| Variable | Value |
|----------|-------|
| `${SSH_PUBLIC_KEY}` | SSH public keys of the first user (fetched from GitHub) |
| `${SSH_PUBLIC_KEY_USERNAME}` | SSH public keys for a specific user (uppercase username) |

Example with multiple users:

```yaml
users:
  - name: ubuntu
    ssh_authorized_keys:
      - ${SSH_PUBLIC_KEY}
  - name: deploy
    ssh_authorized_keys:
      - ${SSH_PUBLIC_KEY_DEPLOY}
```

```json
{
  "vm": {
    "users": [
      {"username": "ubuntu", "github_username": "alice"},
      {"username": "deploy", "github_username": "bot-account"}
    ]
  }
}
```

## DNS Swap (Blue-Green Deployment)

Deploy a new server alongside the old one, then atomically switch DNS:

```bash
goloo create web-server-v2 --aws
# test web-server-v2 ...
goloo dns swap web-server-v2
# web.example.com now points at the new server
goloo delete web-server-v1
```

## Building from Source

Requires Go 1.21+.

```bash
make build          # Build binary to ./bin/goloo
make run-tests      # Run all tests
make clean          # Remove build artifacts
make install        # Install to ~/bin
```

## Prerequisites

- **Local VMs**: [Multipass](https://multipass.run/) installed
- **AWS VMs**: AWS credentials configured (`aws configure`)
- **DNS**: A Route53 hosted zone for your domain (only if using dns section)

## References

- [docs/DESIGN.md](docs/DESIGN.md) — Architecture and diagrams
- [docs/PLAN.md](docs/PLAN.md) — Development phases
- [docs/goloo-notes.md](docs/goloo-notes.md) — Detailed design notes
- [docs/about-the-name.md](docs/about-the-name.md) — Why "goloo"
