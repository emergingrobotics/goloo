# Development Server Example

A general-purpose development VM with build tools, modern CLI utilities, and enough resources for compiling and running services.

## What gets installed

- build-essential (gcc, make, etc.)
- ripgrep, fd-find, fzf, tmux
- git, curl, wget, jq, tree
- vim, htop, unzip

## Before you start

1. [Install Multipass](https://multipass.run/)
2. Build goloo: `make build` from the project root
3. Edit `config.json` and replace `your-github-username` with your actual GitHub username

## Create the VM

From the project root:

```bash
mkdir -p stacks/dev
cp examples/dev-server/config.json examples/dev-server/cloud-init.yaml stacks/dev/

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

Edit `stacks/dev/config.json`:

```json
{
  "vm": {
    "name": "dev",
    "cpus": 4,
    "memory": "4G",
    "disk": "40G",
    "image": "24.04",
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

For heavier workloads, increase the VM resources in `stacks/dev/config.json`:

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
├── README.md
├── config.json              # Config referencing t3.medium for AWS
└── cloud-init.yaml          # Dev tools: build-essential, ripgrep, tmux, etc.
```
