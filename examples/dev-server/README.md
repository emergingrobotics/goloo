# Development Server Example

A general-purpose development VM with build tools, modern CLI utilities, and enough resources for compiling and running services.

## What gets installed

Uses the `configs/dev.yaml` cloud-init from the project root, which includes:

- build-essential (gcc, make, etc.)
- ripgrep, fd-find, fzf, tmux
- git, curl, wget, jq, tree
- vim, htop, unzip

## Before you start

1. [Install Multipass](https://multipass.run/)
2. Build goloo: `make build` from the project root
3. Edit `stacks/dev.json` and replace `your-github-username` with your actual GitHub username

## Create the VM

From the project root:

```bash
cp examples/dev-server/stacks/dev.json stacks/

goloo create dev
```

This creates a VM with 4 CPUs, 4G RAM, and 40G disk -- enough for most development work.

## Verify it worked

```bash
goloo ssh dev
```

Check that the tools are available:

```bash
which rg
which fzf
gcc --version
git --version
```

## Mount your project directory

If you want to edit files on the host and compile in the VM, add a mount to the config before creating:

Edit `stacks/dev.json`:

```json
{
  "vm": {
    "name": "dev",
    "cpus": 4,
    "memory": "4G",
    "disk": "40G",
    "image": "24.04",
    "cloud_init_file": "configs/dev.yaml",
    "mounts": [
      {"source": "/path/to/your/project", "target": "/home/ubuntu/project"}
    ],
    "users": [
      {"username": "ubuntu", "github_username": "your-github-username"}
    ]
  }
}
```

## Adjust resources

For heavier workloads, increase the VM resources in `stacks/dev.json`:

```json
{
  "vm": {
    "cpus": 8,
    "memory": "8G",
    "disk": "80G"
  }
}
```

## Stop and clean up

```bash
goloo stop dev
goloo start dev

goloo delete dev
```

## Files

```
dev-server/
└── stacks/
    └── dev.json                 # Config referencing configs/dev.yaml
```

The cloud-init file (`configs/dev.yaml`) lives in the project root since it's a shared config usable by any example.
