# Web Server Example

An nginx web server with TLS (certbot), intrusion prevention (fail2ban), and a firewall (ufw).

## What gets installed

- nginx
- certbot with nginx plugin
- fail2ban
- ufw (configured to allow SSH, HTTP, and HTTPS)

## Before you start

1. [Install Multipass](https://multipass.run/)
2. Build goloo: `make build` from the project root
3. Edit `config.json` and replace `your-github-username` with your actual GitHub username (this is how goloo fetches your SSH public keys)

## Create the VM

From the project root:

```bash
mkdir -p stacks/web-server
cp examples/web-server/config.json examples/web-server/cloud-init.yaml stacks/web-server/

goloo create web-server
```

## Verify it worked

```bash
goloo status web-server
```

SSH in and check that nginx is running:

```bash
goloo ssh web-server
systemctl status nginx
curl http://localhost
```

You should see the default nginx welcome page.

## Test from the host

Get the VM's IP:

```bash
goloo status web-server
```

Then from your host machine:

```bash
curl http://<vm-ip>
```

## Stop and clean up

```bash
goloo stop web-server      # stop the VM (can restart later)
goloo start web-server     # restart it

goloo delete web-server    # permanently delete
```

## Deploy to AWS instead

The same config file works for both local and AWS. Edit `stacks/web-server/config.json`:
- Set `dns.domain` to a Route53 hosted zone you control
- Set `dns.hostname` to the subdomain you want

Then:

```bash
goloo create web-server --aws
```

## Files

```
web-server/
├── README.md
├── config.json            # Config for local and AWS
└── cloud-init.yaml        # cloud-init: nginx, certbot, fail2ban, ufw
```
