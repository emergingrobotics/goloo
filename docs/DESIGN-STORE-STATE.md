# Design: State Store with Config Snapshots

## Problem

Goloo config files (`config.json` and `cloud-init.yaml`) live wherever the user creates them -- in project directories, in shared repos, or in the default `stacks/` folder. This creates several problems:

1. **`goloo list` is unreliable.** It can only scan one directory. VMs created with `-f /some/other/path` are invisible.
2. **`goloo delete` requires remembering where the config lives.** If a user created a VM six months ago from a project directory, they need to remember (or find) that directory to delete it.
3. **Config files mix specification with runtime state.** After `goloo create --aws`, the config file is mutated with instance IDs, stack names, IPs, and DNS records. This makes configs hard to version control and share.
4. **No history.** Once a VM is deleted, all record of it is gone.

## Design

### Principle

**Configs belong to the user. State belongs to goloo.**

Users author config files wherever makes sense for their workflow -- in project repos, in shared team repos, in `stacks/`. Goloo maintains a separate state directory that it fully owns. This state directory is the single source of truth for "what VMs exist and how to manage them."

### State Directory Layout

```
~/.local/share/goloo/
├── active/                        # Currently running VMs
│   ├── devbox/
│   │   ├── state.json             # Provider state (instance IDs, IPs, stack names)
│   │   ├── config.json            # Snapshot copy of the config used at creation time
│   │   └── cloud-init.yaml        # Snapshot copy of the cloud-init used at creation time
│   └── staging/
│       ├── state.json
│       ├── config.json
│       └── cloud-init.yaml
│
└── archive/                       # Destroyed VMs (kept for reference)
    ├── devbox-2025-01-15T103000/
    │   ├── state.json
    │   ├── config.json
    │   └── cloud-init.yaml
    └── staging-2025-03-20T141500/
        ├── state.json
        ├── config.json
        └── cloud-init.yaml
```

The base path follows the XDG Base Directory Specification:

- Linux: `~/.local/share/goloo/`
- macOS: `~/.local/share/goloo/` (or `~/Library/Application Support/goloo/` if preferred)
- Override: `GOLOO_STATE_DIR` environment variable

### state.json

This file contains only provider-written runtime state. It is never authored by the user.

```json
{
  "name": "devbox",
  "provider": "aws",
  "createdAt": "2025-01-15T10:30:00Z",
  "sourceConfigPath": "/home/user/projects/infra/stacks/devbox",
  "local": {
    "ip": "10.75.123.45"
  },
  "aws": {
    "stackName": "goloo-devbox",
    "stackID": "arn:aws:cloudformation:...",
    "instanceID": "i-0abc123def456",
    "securityGroupID": "sg-0abc123",
    "publicIP": "54.210.33.100",
    "fqdn": "devbox.example.com",
    "zoneID": "Z1234567890",
    "region": "us-east-1",
    "vpcID": "vpc-abc123",
    "subnetID": "subnet-abc123",
    "createdVPC": false,
    "dnsRecords": ["devbox.example.com"]
  }
}
```

- `sourceConfigPath`: Records where the original config was loaded from. Informational only -- never used to read back.
- `local` or `aws`: Only one is populated, depending on the provider.

### Config and Cloud-Init Snapshots

The `config.json` and `cloud-init.yaml` files in the state directory are **copies** made at creation time. They are:

- **Never read back as authoritative config.** If a user runs `goloo create devbox` again, goloo reads from the user's source config, not from the state directory.
- **Snapshots for reference.** They record exactly what spec and cloud-init were used to create this VM. Useful for debugging, auditing, and recreating.
- **Preserved in the archive.** After deletion, the snapshot shows what the destroyed VM looked like.

This means the user's source config files are never touched by goloo's state management. They remain clean, version-controllable, and shareable.

## Operations

### Create

```
goloo create devbox
goloo create devbox --aws
goloo create devbox -f ~/projects/infra
```

**Flow:**

1. Load config from user's source path (current behavior, unchanged)
2. Load cloud-init from user's source path (current behavior, unchanged)
3. Process cloud-init (SSH key substitution, etc.)
4. Call provider to create the VM
5. **New:** Create `~/.local/share/goloo/active/devbox/`
6. **New:** Copy `config.json` into state directory (snapshot)
7. **New:** Copy `cloud-init.yaml` into state directory (snapshot, if it exists)
8. **New:** Write `state.json` with provider state and metadata
9. Save updated config to user's source path (existing behavior -- keeps backward compatibility during transition)

Step 9 maintains backward compatibility. The user's config still gets updated with state as it does today, but the state directory is now the canonical location for that data. Step 9 can be removed in a future version.

**Error: name already active.** If `active/devbox/` already exists, refuse to create. The user must delete or rename first.

### List

```
goloo list
```

**Flow:**

1. **New:** Read all subdirectories of `~/.local/share/goloo/active/`
2. For each, read `state.json`
3. Display VM name, provider, IP, status, creation time

This replaces the current behavior of scanning the stacks directory or shelling out to `multipass list`. The state directory is the authoritative list of goloo-managed VMs.

Optional flags:

- `goloo list --archive` -- also show destroyed VMs from the archive
- `goloo list --all` -- show both active and archived

### Delete

```
goloo delete devbox
```

**Flow:**

1. **New:** Read `state.json` from `~/.local/share/goloo/active/devbox/`
2. Determine provider from state
3. Call provider's delete (using state for AWS instance IDs, stack names, DNS records, etc.)
4. **New:** Move `active/devbox/` to `archive/devbox-{timestamp}/`
5. Update user's source config if it still exists (backward compatibility, remove state fields)

The archive move is the last step. If the provider delete fails, the VM stays in `active/` so the user can retry.

**Timestamp format:** ISO 8601 with no colons or dashes in the time portion for filesystem compatibility: `devbox-20250115T103000`.

**No `-f` flag needed.** The state directory already knows about the VM. The user never needs to remember where the config came from.

### Status

```
goloo status devbox
```

**Flow:**

1. Read `state.json` from `active/devbox/`
2. Optionally query the provider for live status (running/stopped)
3. Display state info alongside live status

### SSH

```
goloo ssh devbox
```

**Flow:**

1. Read `state.json` from `active/devbox/` to get the IP
2. SSH to the IP (current behavior)

### Stop / Start

```
goloo stop devbox
goloo start devbox
```

**Flow:**

1. Read `state.json` from `active/devbox/`
2. Call provider stop/start
3. Update `state.json` if the IP changes (AWS may reassign public IPs on start)

## Archive

The archive serves as a historical record. Destroyed VMs are moved here rather than deleted. This provides:

- **Audit trail.** What VMs existed, when they were created, when they were destroyed, and what config was used.
- **Recovery reference.** If a user accidentally deletes a VM, the config snapshot shows exactly how to recreate it.
- **Debugging.** If AWS resources are left behind (failed cleanup), the archived state has the resource IDs needed to clean up manually.

### Archive Cleanup

The archive grows unbounded by default. Options for managing it:

```bash
goloo archive list                    # List archived VMs
goloo archive clean                   # Delete archives older than 90 days (default)
goloo archive clean --older-than 30d  # Delete archives older than 30 days
goloo archive delete devbox-20250115T103000  # Delete a specific archive entry
```

The archive is just directories on disk. Users can also `rm -rf` entries directly.

## Migration

Existing users have config files with embedded state (AWS fields in `config.json`). Migration happens transparently:

1. On any operation (`list`, `delete`, `status`, etc.), if goloo finds no state directory for a VM but finds a config with provider state, it creates the state directory entry from the existing config.
2. A one-time `goloo migrate` command scans a stacks folder and creates state entries for all VMs found.

```bash
goloo migrate                     # Migrate from default stacks/
goloo migrate -f ~/my-servers     # Migrate from custom path
```

## Configuration

The state directory location can be configured via:

1. `GOLOO_STATE_DIR` environment variable (highest priority)
2. `~/.config/goloo/settings.json` with a `stateDir` field
3. Default: `~/.local/share/goloo/`

```json
{
  "stateDir": "/home/user/custom/goloo-state"
}
```

## Data Flow Diagram

```
                    User's Project
                    ┌──────────────────────┐
                    │ stacks/devbox/       │
                    │ ├── config.json      │  ← User authors and edits these
                    │ └── cloud-init.yaml  │  ← Version controlled, shareable
                    └──────────┬───────────┘
                               │
                     goloo create devbox
                               │
                    ┌──────────▼───────────┐
                    │                      │
                    │   goloo create flow   │
                    │                      │
                    │  1. Read user config  │
                    │  2. Process cloud-init│
                    │  3. Create VM         │
                    │  4. Copy files to     │
                    │     state dir         │
                    │  5. Write state.json  │
                    │                      │
                    └──────────┬───────────┘
                               │
              ~/.local/share/goloo/
              ┌────────────────▼───────────────────┐
              │ active/devbox/                      │
              │ ├── state.json        ← goloo owns │
              │ ├── config.json       ← snapshot   │
              │ └── cloud-init.yaml   ← snapshot   │
              │                                    │
              │ archive/devbox-20250115T103000/     │
              │ ├── state.json        ← preserved  │
              │ ├── config.json       ← preserved  │
              │ └── cloud-init.yaml   ← preserved  │
              └────────────────────────────────────┘

         goloo list    → reads active/**/state.json
         goloo delete  → reads active/name/state.json, calls provider, moves to archive/
         goloo status  → reads active/name/state.json
         goloo ssh     → reads active/name/state.json for IP
```

## File Ownership Rules

| File | Who writes | Who reads | Location |
|------|-----------|-----------|----------|
| User's `config.json` | User | goloo (on create) | User's project |
| User's `cloud-init.yaml` | User | goloo (on create) | User's project |
| State `state.json` | goloo | goloo | State directory |
| Snapshot `config.json` | goloo (copy) | User (reference only) | State directory |
| Snapshot `cloud-init.yaml` | goloo (copy) | User (reference only) | State directory |

## Scope

### In scope for initial implementation

- State directory creation and management (`active/`, `archive/`)
- `state.json` writing on create, reading on all operations
- Config and cloud-init snapshot copying on create
- Archive move on delete
- `goloo list` reading from state directory
- `goloo migrate` for existing users
- `GOLOO_STATE_DIR` environment variable

### Out of scope (future)

- `goloo archive` subcommands (list, clean, delete)
- Settings file (`~/.config/goloo/settings.json`)
- Removing backward-compatible state writes to user's config
- Remote state backends (S3, etc.)

## Implementation Notes

### New package: `internal/store`

Responsible for all state directory operations:

```go
package store

type Store struct {
    BaseDir string // e.g. ~/.local/share/goloo
}

func New() (*Store, error)                                      // Resolve base dir, create if needed
func (s *Store) SaveState(name string, state *State) error      // Write state.json
func (s *Store) LoadState(name string) (*State, error)          // Read state.json
func (s *Store) CopyConfig(name string, configPath string) error // Copy config.json snapshot
func (s *Store) CopyCloudInit(name string, ciPath string) error  // Copy cloud-init.yaml snapshot
func (s *Store) ListActive() ([]string, error)                  // List active VM names
func (s *Store) Archive(name string) error                      // Move active to archive
func (s *Store) Exists(name string) bool                        // Check if VM is active
func (s *Store) Migrate(folder string) error                    // Import existing configs
```

### State struct

```go
type State struct {
    Name             string     `json:"name"`
    Provider         string     `json:"provider"`
    CreatedAt        time.Time  `json:"createdAt"`
    SourceConfigPath string     `json:"sourceConfigPath"`
    Local            *LocalState `json:"local,omitempty"`
    AWS              *AWSState   `json:"aws,omitempty"`
}
```

The `LocalState` and `AWSState` structs match the existing ones in `internal/config/config.go`. They can be shared or duplicated depending on whether the config struct refactor happens.

### Changes to existing code

- `cmd/goloo/main.go`: After create, call store to save state and copy files. On delete, read from store instead of (or in addition to) loading user config. On list, read from store.
- Provider `Delete()` methods: Accept state instead of (or in addition to) config for resource IDs.
- Minimal changes to `internal/config/` -- it continues to load user configs as today.
