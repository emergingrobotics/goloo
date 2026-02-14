# Whole-Server Configuration: Options for Goloo

How to describe all the software on a VM and how it's configured to work together.

## The Problem

Installing packages is easy — cloud-init's `packages:` list handles that. The hard part is **wiring**: nginx needs to know what port the app listens on, the app needs the database connection string, the database needs to be initialized with the right user and schema, and TLS certificates need the correct domain. Today, goloo users write all of this by hand in cloud-init YAML.

The question: what's the right way for goloo to help users describe a complete, integrated server?

## Where Goloo Is Today

Goloo already has building blocks:

- **Go template rendering** in cloud-init YAML with a `TemplateData` struct exposing VM name, DNS config, users, SSH keys, packages, and an arbitrary `Vars` map
- **Config-driven packages** via the `cloud_init.packages` field
- **Custom variables** via the `cloud_init.vars` map, accessible as `{{ .Vars.key }}` in templates
- **Template library** in `configs/` with pre-built cloud-init files (`base.yaml`, `dev.yaml`, `go-dev.yaml`, etc.)

These are the foundation any approach builds on.

---

## Approach 1: Composable Cloud-Init Fragments

Merge multiple cloud-init YAML files into one, each handling a different concern (base packages, nginx, postgres, app deployment).

### How It Works

Cloud-init supports merging via multi-part MIME messages. Each part is a `text/cloud-config` document. Cloud-init merges them in order using configurable rules.

Goloo would have a fragment library:

```
fragments/
├── base.yaml          # curl, git, htop, ufw
├── nginx.yaml         # nginx + config
├── postgres.yaml      # postgresql + initial db
├── nodejs.yaml        # node.js runtime
├── certbot.yaml       # let's encrypt
└── monitoring.yaml    # prometheus node_exporter
```

Config selects fragments:

```json
{
  "cloud_init": {
    "fragments": ["base", "nginx", "postgres", "nodejs", "certbot"]
  }
}
```

Goloo merges them in Go (not relying on cloud-init's MIME merge) and produces a single cloud-init YAML.

### The Wiring Problem

Poorly handled. Fragments are isolated. The nginx fragment can't know the app runs on port 3000 unless you add a variable substitution pass on top. At that point you're building Approach 6 anyway.

### Verdict

Simple to implement but doesn't solve the hard problem. Works well for additive concerns (install this, install that) but breaks down when services need to reference each other.

---

## Approach 2: Declarative Service Manifest

A new config layer where users declare services and their relationships. Goloo generates the cloud-init YAML.

### How It Works

```yaml
services:
  reverse-proxy:
    type: caddy
    config:
      domain: "{{ dns.fqdn }}"
      upstream_port: "{{ services.app.port }}"

  app:
    type: nodejs
    config:
      port: 3000
      env:
        DATABASE_URL: "postgresql://{{ services.db.user }}:{{ services.db.password }}@localhost:{{ services.db.port }}/{{ services.db.database }}"

  db:
    type: postgresql
    config:
      port: 5432
      database: myapp
      user: appuser
      password: "{{ secrets.db_password }}"
```

Goloo resolves cross-references, loads a template for each service type, and assembles the cloud-init YAML. Since everything runs on one VM, inter-service references resolve to `localhost:PORT`.

A service type library defines what each type installs and how it's configured:

```
service-types/
├── caddy/
│   ├── schema.json       # required/optional config fields
│   └── template.yaml     # cloud-init stanzas this service produces
├── nginx/
├── postgresql/
├── nodejs/
├── redis/
└── docker/
```

### The Wiring Problem

Solved explicitly. `{{ services.db.port }}` references are first-class. Goloo validates the service graph before generating anything — if the app references a service that doesn't exist, you get an error, not a broken VM.

### What This Looks Like for a Real Server

A web application with Caddy, a Node.js app, PostgreSQL, and Redis:

```json
{
  "vm": {
    "name": "prod-web",
    "cpus": 4,
    "memory": "4G",
    "disk": "40G",
    "image": "24.04",
    "instance_type": "t3.medium",
    "users": [{"username": "ubuntu", "github_username": "alice"}]
  },
  "dns": {
    "hostname": "app",
    "domain": "example.com"
  },
  "services": {
    "proxy": {
      "type": "caddy",
      "domain": "app.example.com",
      "upstream": 3000
    },
    "app": {
      "type": "nodejs",
      "port": 3000,
      "repo": "https://github.com/myorg/myapp.git",
      "env": {
        "DATABASE_URL": "postgresql://appuser:${DB_PASSWORD}@localhost:5432/myapp",
        "REDIS_URL": "redis://localhost:6379"
      }
    },
    "db": {
      "type": "postgresql",
      "database": "myapp",
      "user": "appuser"
    },
    "cache": {
      "type": "redis"
    }
  }
}
```

### Verdict

The most user-friendly approach and the best solution for the wiring problem. But it's the largest implementation effort — essentially building a service-type library with templates, a dependency resolver, a cross-reference engine, and a validation layer. The service-type library also needs ongoing maintenance as upstream software changes.

---

## Approach 3: Ansible Integration

Cloud-init bootstraps Ansible on the VM and runs a playbook that handles all configuration.

### How It Works

Cloud-init has a native `cc_ansible` module (since cloud-init 22.3):

```yaml
#cloud-config
ansible:
  install_method: pip
  pull:
    - url: "https://github.com/myorg/server-playbooks.git"
      playbook_names: [webserver.yml]
```

Cloud-init installs Ansible, clones the repo, and runs the playbook locally. The playbook uses roles:

```yaml
# webserver.yml
- hosts: localhost
  roles:
    - role: base
    - role: postgresql
      vars:
        db_name: myapp
        db_user: appuser
    - role: nodejs
      vars:
        app_repo: https://github.com/myorg/myapp.git
        app_port: 3000
    - role: caddy
      vars:
        domain: app.example.com
        upstream_port: 3000
```

Goloo's config would reference the playbook repo and pass variables:

```json
{
  "cloud_init": {
    "ansible": {
      "repo": "https://github.com/myorg/server-playbooks.git",
      "playbook": "webserver.yml",
      "vars": {
        "db_name": "myapp",
        "app_port": 3000,
        "domain": "app.example.com"
      }
    }
  }
}
```

### The Wiring Problem

Handled well. Ansible roles pass variables to each other. A top-level playbook defines `vars` consumed by multiple roles. Jinja2 templates within roles can reference any variable in scope. Ansible Galaxy has thousands of pre-built roles.

### Trade-offs

- Adds 2-5 minutes to first boot (installing Ansible + Python dependencies)
- Requires hosting playbooks somewhere accessible (GitHub works for public repos; private repos need credential bootstrapping)
- Users must know Ansible or adopt it
- Two-layer config: goloo's `config.json` + Ansible playbooks/roles
- Debugging ansible-pull failures on a VM with no interactive terminal is painful

### Verdict

The pragmatic "outsource the hard part" option. Goloo stays thin (just generates the ansible bootstrap cloud-init), and Ansible handles all the complex configuration logic. Good if your team already uses Ansible. Not great if they don't — the learning curve is significant and the first-boot overhead is real.

---

## Approach 4: Shell Script Bundles

Cloud-init writes modular shell scripts to disk and executes them in order.

### How It Works

```yaml
write_files:
  - path: /opt/setup/01-base.sh
    permissions: '0755'
    content: |
      #!/bin/bash
      set -euo pipefail
      apt-get update && apt-get install -y curl git htop ufw

  - path: /opt/setup/02-postgres.sh
    permissions: '0755'
    content: |
      #!/bin/bash
      set -euo pipefail
      apt-get install -y postgresql
      sudo -u postgres createuser appuser
      sudo -u postgres createdb -O appuser myapp

  - path: /opt/setup/03-app.sh
    permissions: '0755'
    content: |
      #!/bin/bash
      set -euo pipefail
      curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
      apt-get install -y nodejs
      # clone and start app...

runcmd:
  - for script in /opt/setup/*.sh; do bash "$script"; done
```

Goloo would maintain a library of script fragments and assemble selected ones into `write_files` entries.

### The Wiring Problem

Manual. Each script writes config files with hardcoded values. You generate `/etc/caddy/Caddyfile` with the right domain and upstream port inside the script. Fragile, but transparent.

### Verdict

This is what goloo effectively does today — the existing `configs/*.yaml` templates are shell-script-based cloud-init files. It works for simple cases but doesn't scale to complex multi-service setups. No dependency resolution, no validation, not idempotent.

---

## Approach 5: Extended Template Generation

Push goloo's existing Go template system further with composable partials and richer variable passing.

### How It Works

Goloo already renders Go templates with `TemplateData`. Extend this with:

**Template partials** — reusable snippets that can be included:

```yaml
#cloud-config
packages:
  - curl
  - git
  {{ template "postgres-packages" . }}
  {{ template "nodejs-packages" . }}

write_files:
  {{ template "postgres-config" . }}
  {{ template "caddy-config" . }}

runcmd:
  {{ template "postgres-init" . }}
  {{ template "nodejs-setup" . }}
  {{ template "caddy-setup" . }}
```

**Richer vars** in config for wiring:

```json
{
  "cloud_init": {
    "template": "web-app-server",
    "vars": {
      "app_port": 3000,
      "db_name": "myapp",
      "db_user": "appuser",
      "cache_enabled": true
    }
  }
}
```

Templates use these to generate inter-service config:

```yaml
# caddy-config partial
- path: /etc/caddy/Caddyfile
  content: |
    {{ .Vars.domain }} {
        reverse_proxy localhost:{{ .Vars.app_port }}
    }
```

### The Wiring Problem

Partially solved. Variables carry port numbers, database names, and domains between template partials. But the wiring is manual — the user must set the right variables, and there's no validation that `app_port` in the Caddy config matches the port the app actually listens on.

### Verdict

The smallest step from where goloo is today. Incremental, low-risk, and immediately useful. Doesn't fully solve the wiring problem but makes it manageable for common cases. The template library grows organically — each new partial handles one service.

---

## Approach 6: Nix-Style Declarative System Config

Declare the entire system state in a functional language. The tool generates everything needed to match that declaration.

### How It Works

NixOS does this with `configuration.nix`:

```nix
{
  services.nginx.enable = true;
  services.nginx.virtualHosts."app.example.com" = {
    enableACME = true;
    locations."/".proxyPass = "http://localhost:3000";
  };
  services.postgresql.enable = true;
  services.postgresql.ensureDatabases = [ "myapp" ];
}
```

For goloo, this would mean either targeting NixOS images (breaking the Ubuntu/cloud-init model) or building a Nix-inspired DSL that generates cloud-init YAML.

### Verdict

Architecturally incompatible with goloo. NixOS requires NixOS images, which Multipass doesn't support. A Nix-inspired DSL would be Approach 2 with extra complexity. The learning curve is steep and the ecosystem is niche.

---

## Comparison

| Approach | Implementation Effort | Wiring | User Complexity | Fits Goloo |
|---|---|---|---|---|
| 1. Cloud-init fragments | Low | Poor | Medium | Yes |
| 2. Service manifest | High | Excellent | Low | Yes |
| 3. Ansible integration | Low (Go side) | Good | High (Ansible) | Yes |
| 4. Shell script bundles | Low | Manual | Low | Already doing this |
| 5. Extended templates | Low-Medium | Partial | Low-Medium | Yes, incremental |
| 6. Nix-style | Very High | Excellent | Very High | No |

---

## Recommendation: Start with Approach 5, Grow Toward Approach 2

### Phase 1: Extended Templates (now)

Build on what exists. Add template partials for common services and document the `cloud_init.vars` pattern for wiring. This is a small code change with immediate value.

A user wanting "Caddy + Node.js + PostgreSQL" would use a `web-app` template and set variables in their config:

```json
{
  "vm": { "name": "prod", "..." : "..." },
  "dns": { "hostname": "app", "domain": "example.com" },
  "cloud_init": {
    "template": "web-app",
    "vars": {
      "app_port": 3000,
      "app_repo": "https://github.com/myorg/myapp.git",
      "db_name": "myapp",
      "db_user": "appuser"
    }
  }
}
```

The `web-app` template is a cloud-init YAML that uses Go template syntax to wire everything together. Goloo ships several of these templates for common patterns (web app, API server, dev environment).

### Phase 2: Service Types (later, if needed)

If the template approach hits limits — too many variables, too much copy-paste between templates, wiring errors that aren't caught — evolve toward Approach 2. Introduce a `services` section in config where each service has a type and parameters. Goloo validates the service graph and generates the cloud-init YAML.

The migration path is smooth: service types are internally just parameterized templates with a schema. The user-facing config gets cleaner, but the generation mechanism is the same.

### Why Not Ansible?

Ansible is the right answer for teams that already use Ansible. For goloo's target user — a developer who wants to spin up a configured VM without learning a new tool — adding an Ansible dependency doubles the concepts they need to understand. The 2-5 minute first-boot overhead is also significant for the local development use case where you want a VM running quickly.

### Why Not Fragments?

Fragments solve installation but not wiring. If all you need is "install these packages," cloud-init's `packages:` list already works. The hard problem is configuration, and fragments don't help there.

---

## The Wiring Problem in Detail

For any approach, here's what "wiring" concretely means on a single VM:

| Service A | Needs to Know | From Service B |
|---|---|---|
| Caddy/Nginx | upstream port | App |
| Caddy/Nginx | domain name | DNS config |
| App | database connection string | PostgreSQL |
| App | redis URL | Redis |
| App | environment variables | Secrets/config |
| PostgreSQL | which database to create | App |
| PostgreSQL | which user to create | App |
| Certbot/ACME | domain name | DNS config |
| UFW/firewall | which ports to open | All services |
| Systemd | service dependencies | All services |

Since everything runs on `localhost`, the wiring reduces to:
1. **Port numbers** — who listens where
2. **Credentials** — database passwords, API keys
3. **File paths** — where configs and sockets live
4. **Domain names** — for TLS and virtual hosting
5. **Startup ordering** — systemd `After=` and `Requires=`

Any solution must handle these five categories. The template approach handles them through variables. The service manifest approach handles them through cross-references. Ansible handles them through role variables. Shell scripts handle them through hardcoded values.

---

## Comparable Tools

| Tool | Approach | Target |
|---|---|---|
| Docker Compose | Service manifest with networking | Containers |
| Kamal | Docker deployment with env var wiring | Remote servers |
| Dokku | Plugin-based services with auto-linking | Single server PaaS |
| CapRover | Docker Swarm with one-click app templates | Single server PaaS |
| Packer | Image building with provisioners | Machine images |
| Vagrant | VM provisioning with pluggable provisioners | Local development |
| Helm | Parameterized templates with values | Kubernetes |

Goloo's closest analog is **Vagrant with provisioners** for local and **Packer + cloud-init** for cloud. The template approach (Approach 5) is closest to **Helm's model** — parameterized templates with a values file.

The service manifest approach (Approach 2) would make goloo closer to **Docker Compose for bare metal** — declare what you want, the tool figures out how to wire it.
