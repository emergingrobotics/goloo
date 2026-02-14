# Configure Apps: Post-Install Script Injection

## Overview

Goloo injects a standalone `configure.sh` script from a stack folder into the cloud-init `write_files` section at VM creation time. This lets you write configuration scripts as normal shell files while goloo handles embedding them into cloud-init automatically.

## Stack Folder Layout

```
stacks/blog2/
├── config.json
├── cloud-init.yaml
└── configure.sh          ← optional post-install script
```

If `configure.sh` is present in the stack folder, goloo injects it. If absent, cloud-init passes through unchanged (existing behavior).

## How It Works

### 1. Author the cloud-init file

The `cloud-init.yaml` declares packages and any other cloud-init directives as usual. It does NOT need to reference `configure.sh` — goloo handles that.

```yaml
#cloud-config
users:
  - name: ubuntu
    ssh_authorized_keys:
      - ${SSH_PUBLIC_KEY}

packages:
  - caddy
  - postgresql
  - redis
```

### 2. Author the configure script

`configure.sh` is a standard bash script. It runs as root during the cloud-init final stage, after all packages have been installed.

```bash
#!/bin/bash
set -euo pipefail

# Configure Caddy
cat > /etc/caddy/Caddyfile <<'CADDY'
:80 {
    root * /var/www/html
    file_server
}
CADDY
systemctl reload caddy

# Configure Postgres
sudo -u postgres createdb myapp
sudo -u postgres psql -c "CREATE USER app WITH PASSWORD 'secret';"

# Configure Redis
sed -i 's/^bind 127.0.0.1/bind 0.0.0.0/' /etc/redis/redis.conf
systemctl restart redis
```

### 3. Goloo injects at create time

When `goloo create blog2` runs, goloo:

1. Loads `cloud-init.yaml` from the stack folder
2. Performs existing SSH key substitution (`${SSH_PUBLIC_KEY}`)
3. Checks for `configure.sh` in the same folder
4. If found, reads its contents and appends a `write_files` entry and a `runcmd` entry to the cloud-init YAML
5. Passes the assembled YAML to multipass or AWS

### 4. What goloo generates

Given the above inputs, goloo produces this final cloud-init YAML (passed to the provider, never written to disk):

```yaml
#cloud-config
users:
  - name: ubuntu
    ssh_authorized_keys:
      - ssh-ed25519 AAAAC3... (fetched from GitHub)

packages:
  - caddy
  - postgresql
  - redis

write_files:
  - path: /opt/goloo/configure.sh
    permissions: '0755'
    content: |
      #!/bin/bash
      set -euo pipefail

      # Configure Caddy
      cat > /etc/caddy/Caddyfile <<'CADDY'
      :80 {
          root * /var/www/html
          file_server
      }
      CADDY
      systemctl reload caddy

      # Configure Postgres
      sudo -u postgres createdb myapp
      sudo -u postgres psql -c "CREATE USER app WITH PASSWORD 'secret';"

      # Configure Redis
      sed -i 's/^bind 127.0.0.1/bind 0.0.0.0/' /etc/redis/redis.conf
      systemctl restart redis

runcmd:
  - /opt/goloo/configure.sh
```

## Cloud-Init Execution Order

This ordering is guaranteed by cloud-init's stage architecture:

```
Stage        What Runs                          Order
───────────────────────────────────────────────────────
local        networking setup                    1
network      fetch metadata, write_files         2
config       packages (apt update/install)       3
final        runcmd (/opt/goloo/configure.sh)    4
```

The configure script always runs after packages are fully installed because `runcmd` executes in the final stage while `packages` executes in the config stage.

## Implementation Details

### Injection logic

The injection belongs in `internal/cloudinit/`, alongside the existing SSH key substitution. The processing pipeline becomes:

```
cloud-init.yaml (raw)
    │
    ├── substitute ${SSH_PUBLIC_KEY} with GitHub keys
    │
    ├── if configure.sh exists:
    │   ├── parse YAML
    │   ├── append to write_files list (create if absent)
    │   ├── append to runcmd list (create if absent)
    │   └── serialize back to YAML
    │
    └── final cloud-init YAML → provider (multipass or AWS)
```

### YAML merging rules

- If `write_files` already exists in the cloud-init YAML, append the configure script entry to the existing list
- If `write_files` does not exist, create it with the single entry
- Same logic for `runcmd` — append `/opt/goloo/configure.sh` to any existing commands
- User-defined `runcmd` entries run first (preserve ordering), goloo's entry appends last
- This ensures the user can have their own `write_files` and `runcmd` entries without conflict

### Script destination path

Scripts are placed at `/opt/goloo/configure.sh` on the VM. Using `/opt/goloo/` as the namespace avoids collisions with system paths and makes it clear where the script came from.

### Content encoding

The script content is embedded using YAML literal block scalar (`|`). This preserves newlines and avoids escaping issues. The YAML serializer must indent the script content correctly relative to the `content` key.

### Edge cases

| Case | Behavior |
|------|----------|
| No `configure.sh` in stack folder | No injection, existing behavior unchanged |
| Empty `configure.sh` | Skip injection (treat as absent) |
| `cloud-init.yaml` already has `write_files` | Append to existing list |
| `cloud-init.yaml` already has `runcmd` | Append to existing list |
| No `cloud-init.yaml` in stack folder | No injection (no cloud-init to inject into) |
| `configure.sh` contains `${SSH_PUBLIC_KEY}` | SSH substitution runs first, so it gets replaced here too |

## Affected Code

```
internal/cloudinit/
├── cloudinit.go          ← add InjectConfigureScript()
└── cloudinit_test.go     ← test injection, merging, edge cases

internal/provider/
├── multipass/multipass.go  ← no change (receives final YAML)
└── aws/aws.go              ← no change (receives final YAML)
```

The providers don't change. They already receive processed cloud-init YAML — the injection happens upstream in the cloudinit package before the YAML reaches any provider.

## Testing

Unit tests should cover:

- Injection when `configure.sh` is present
- No injection when `configure.sh` is absent
- Merging with existing `write_files` entries
- Merging with existing `runcmd` entries
- Empty `configure.sh` is skipped
- Script content with special characters (quotes, heredocs, dollar signs) survives YAML encoding
- SSH key substitution and script injection compose correctly

## Example Workflow

```bash
# Create the stack
mkdir -p stacks/blog2

# Write config
cat > stacks/blog2/config.json <<'EOF'
{
  "vm": {
    "name": "blog2",
    "cpus": 2,
    "memory": "2G",
    "disk": "20G",
    "image": "24.04",
    "users": [
      {"username": "ubuntu", "github_username": "gherlein"}
    ]
  }
}
EOF

# Write cloud-init (packages only)
cat > stacks/blog2/cloud-init.yaml <<'EOF'
#cloud-config
users:
  - name: ubuntu
    ssh_authorized_keys:
      - ${SSH_PUBLIC_KEY}

packages:
  - caddy
EOF

# Write configure script (post-install config)
cat > stacks/blog2/configure.sh <<'EOF'
#!/bin/bash
set -euo pipefail

cat > /etc/caddy/Caddyfile <<'CADDY'
:80 {
    root * /var/www/html
    file_server
}
CADDY

systemctl reload caddy
EOF

# Launch — goloo handles the rest
goloo create blog2

# Verify
goloo ssh blog2
curl localhost    # Caddy responds on port 80, no HTTPS redirect
```
