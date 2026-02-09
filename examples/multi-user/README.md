# Multi-User Example

A VM with two separate user accounts, each with their own SSH keys pulled from different GitHub accounts. Useful for shared servers where a human developer and a deployment bot both need access.

## What gets installed

- curl, wget, vim, htop, git

## Users created

| VM user | SSH keys from | Purpose |
|---------|--------------|---------|
| `ubuntu` | First GitHub user (`${SSH_PUBLIC_KEY}`) | Primary developer account |
| `deploy` | Second GitHub user (`${SSH_PUBLIC_KEY_DEPLOY}`) | Deployment/CI bot account |

Both users have passwordless sudo.

## Before you start

1. [Install Multipass](https://multipass.run/)
2. Build goloo: `make build` from the project root
3. Edit `stacks/multi-user.json` and replace the GitHub usernames:
   - Change `alice` to the primary developer's GitHub username
   - Change `bot-account` to the deployment account's GitHub username

Both GitHub accounts need SSH public keys uploaded at `github.com/settings/keys`.

## Create the VM

From the project root:

```bash
cp examples/multi-user/stacks/multi-user.json stacks/
cp examples/multi-user/cloud-init/multi-user.yaml cloud-init/

goloo create multi-user
```

## Verify it worked

SSH in as the primary user:

```bash
goloo ssh multi-user
```

Check that both users exist:

```bash
id ubuntu
id deploy
```

Check that each user has their SSH keys:

```bash
cat ~/.ssh/authorized_keys
sudo cat /home/deploy/.ssh/authorized_keys
```

## SSH as the deploy user

Get the VM's IP:

```bash
goloo status multi-user
```

Then SSH directly (requires the deploy account's private key):

```bash
ssh deploy@<vm-ip>
```

## How the SSH key variables work

In the cloud-init YAML:

- `${SSH_PUBLIC_KEY}` is replaced with the SSH keys of the **first** user listed in the config (ubuntu/alice)
- `${SSH_PUBLIC_KEY_DEPLOY}` is replaced with the SSH keys of the user whose **username** is `deploy` (uppercased in the variable name)

The variable naming pattern is `${SSH_PUBLIC_KEY_<USERNAME>}` where `<USERNAME>` is the VM username in uppercase.

## Stop and clean up

```bash
goloo stop multi-user
goloo start multi-user

goloo delete multi-user
```

## Files

```
multi-user/
├── cloud-init/
│   └── multi-user.yaml          # cloud-init with two users and per-user SSH keys
└── stacks/
    └── multi-user.json           # Config with two users mapped to GitHub accounts
```
