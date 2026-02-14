# Config Storage Options

This document evaluates options for centralizing goloo config file storage. Today, config files (`config.json` and `cloud-init.yaml`) live in `stacks/<name>/` directories that can be anywhere on the filesystem. This creates problems:

- Users forget where they created VMs, making `delete` and `list` unreliable
- The `-f` flag is required to manage VMs created outside the default `stacks/` folder
- `config.json` mixes user-authored specification with mutable provider state (AWS instance IDs, IPs, stack names), making configs hard to version control

## Constraints

Config files serve two purposes:

1. **Input (spec)**: User-authored VM specification (cpus, memory, image, cloud-init template)
2. **State**: Provider writes back runtime state (instance IDs, IPs, stack names) after provisioning

Any solution must handle both. AWS delete requires access to stored state (stack names, DNS records, VPC info) to tear down resources.

---

## Option 1: XDG-Compliant Home Directory

Store everything under a single well-known path following the XDG Base Directory Specification.

```
~/.config/goloo/stacks/
├── devbox/
│   ├── config.json
│   └── cloud-init.yaml
├── staging/
│   ├── config.json
│   └── cloud-init.yaml
```

- `goloo create devbox` looks in `~/.config/goloo/stacks/devbox/` by default
- The `-f` flag still works for one-offs
- `GOLOO_STACK_FOLDER` still works as an override

**Pros:** Simplest change. Single location. Follows OS conventions. The existing code barely changes -- just swap the default from `"stacks"` to `~/.config/goloo/stacks`. macOS/Linux standard.

**Cons:** Still just flat files. No multi-machine or team sharing. Mixing mutable state with user-authored config in the same file means noisy diffs when version controlling configs.

---

## Option 2: Separate Config Templates from State

Split the current `config.json` into two concerns, similar to how Terraform separates `.tf` files from `.tfstate`:

```
~/.config/goloo/templates/         # User-authored, git-trackable
├── devbox/
│   ├── config.json                # VM spec only (cpus, memory, image, dns)
│   └── cloud-init.yaml
│
~/.local/share/goloo/state/        # Runtime state, machine-local
├── devbox.state.json              # {provider: "aws", instanceId: "i-abc", ip: "1.2.3.4", ...}
```

- Templates are immutable from goloo's perspective -- it reads but never writes them
- State files are created/updated by providers after create, cleared on delete
- Templates can live in a shared git repo; state stays local

**Pros:** Clean separation of concerns. Templates become safely git-trackable and shareable. State is clearly machine-local. Matches established patterns (Terraform, Pulumi). AWS delete reads state file for instance IDs, not the template.

**Cons:** More significant refactor -- need to split the `Config` struct into spec vs. state. Two locations to reason about. Migration path needed for existing users.

---

## Option 3: Embedded Database (BoltDB or SQLite)

Store configs and state in a single database file:

```
~/.local/share/goloo/goloo.db
```

A single BoltDB (pure Go, no CGO) or SQLite file holds all VM records:

```
Key: "devbox"
Value: {
  spec: { cpus: 4, memory: "4G", ... },
  cloudInit: "...",
  state: { provider: "aws", instanceId: "i-abc", ... }
}
```

CLI gains management commands:

```bash
goloo config import devbox ./my-configs/devbox/
goloo config export devbox ./backup/
goloo config list
goloo config edit devbox
```

**Pros:** Single file -- easy to back up, move, copy. No scattered directories. Atomic reads/writes. Can store metadata (created-at, last-used, tags). Querying is trivial.

**Cons:** Not human-editable without tooling (need import/export/edit commands). Harder to diff or version control. BoltDB adds a dependency. Cloud-init authoring workflow becomes indirect -- write file, then import it.

---

## Option 4: Remote State Backend (S3)

Store state remotely for team use, keep templates local or in a git repo:

```
Templates: local filesystem or git repo (as today)
State:     s3://my-goloo-state/devbox.state.json
```

Configuration in `~/.config/goloo/settings.json`:

```json
{
  "stateBackend": {
    "type": "s3",
    "bucket": "my-goloo-state",
    "region": "us-east-1",
    "prefix": "team-infra/"
  }
}
```

- On `goloo create`, state is written to S3 after provisioning
- On `goloo delete`, state is read from S3 to get instance IDs, then cleared
- Optional DynamoDB lock table prevents concurrent operations on the same VM

**Pros:** Team-friendly -- anyone with bucket access can manage VMs. State survives laptop loss. Natural fit since goloo already uses AWS. Locking prevents conflicts.

**Cons:** Requires AWS setup even for local-only users. Network dependency for every operation. Adds complexity (auth, bucket policies, error handling for network failures). Overkill for single-user scenarios.

---

## Option 5: Central Registry with Distributed Storage

A lightweight index file tracks all VM configs regardless of where they physically live:

```
~/.config/goloo/registry.json
```

```json
{
  "vms": {
    "devbox": {
      "configPath": "/home/user/projects/infra/stacks/devbox",
      "provider": "multipass",
      "createdAt": "2025-01-15T10:30:00Z"
    },
    "staging": {
      "configPath": "/home/user/work/staging-server",
      "provider": "aws",
      "createdAt": "2025-01-20T14:00:00Z"
    }
  }
}
```

- `goloo create devbox` registers the config path in the registry
- `goloo delete staging` looks up the path from the registry, loads config from there
- `goloo list` reads the registry instead of scanning directories
- Configs stay wherever the user puts them -- the registry just knows where they are

**Pros:** Zero migration -- works with existing file layout. Users organize configs however they want (per-project, shared repo, centralized). `goloo list` and `goloo delete` work without remembering `-f` paths. Very small code change.

**Cons:** Registry can get stale (config moved/deleted without goloo knowing). Two sources of truth (registry says one thing, filesystem says another). Does not solve the template-vs-state mixing problem.

---

## Option 6: State Store with Config Snapshots

Detailed in [DESIGN-STORE-STATE.md](DESIGN-STORE-STATE.md).

Configs stay with the project (wherever the user authors them). On `goloo create`, config and cloud-init files are **copied** into a well-known state directory. The copies are never read back as authoritative config -- they exist as a snapshot of what was used to create the VM, alongside provider state. `goloo list` reads from the state directory. `goloo delete` moves the VM's state folder to an archive.

**Pros:** Configs stay in project repos where they belong. State directory is a complete record of all VMs. List and delete always work without `-f`. Archive provides history. No config struct refactor needed.

**Cons:** Copies can drift from source configs if the user edits after create. Slightly more disk usage (copies are small).

---

## Recommendation

**Option 6 (State Store with Config Snapshots)** is the best balance of simplicity, reliability, and usefulness. It solves the core problems (list/delete reliability, config discoverability) without requiring a config struct refactor or changing the authoring workflow. Configs live wherever the user wants them -- in project repos, in shared git repos, in `stacks/`. The state store is an implementation detail that makes operations reliable.

See [DESIGN-STORE-STATE.md](DESIGN-STORE-STATE.md) for the full design.
