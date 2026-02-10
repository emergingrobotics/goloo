# AWS Web Server Example

An nginx web server deployed to AWS EC2 with TLS (certbot), intrusion prevention (fail2ban), a firewall (ufw), Route53 DNS, and two user accounts.

This is the AWS counterpart to the [web-server](../web-server/) example. Same cloud-init base, full AWS config with all available parameters.

## What gets installed

- nginx, certbot with nginx plugin, fail2ban, ufw
- htop, net-tools, tree, jq, curl, wget, vim, git

## Users created

| VM user | SSH keys from | Purpose |
|---------|---------------|---------|
| `ubuntu` | First GitHub user (`${SSH_PUBLIC_KEY}`) | Primary admin account |
| `deploy` | Second GitHub user (`${SSH_PUBLIC_KEY_DEPLOY}`) | Deployment/CI bot account |

## Config fields

### vm section

| Field | Value | Purpose |
|-------|-------|---------|
| `name` | `aws-web-server` | VM name and CloudFormation stack prefix |
| `instance_type` | `t3.small` | EC2 instance size |
| `os` | `ubuntu-24.04` | AMI lookup key (see supported OS list below) |
| `region` | `us-east-1` | AWS region |
| `vpc_id` | `""` | VPC to deploy into (empty = auto-discover default VPC) |
| `subnet_id` | `""` | Subnet to deploy into (empty = auto-discover) |
| `users` | 2 users | GitHub accounts for SSH key injection |

### dns section

| Field | Value | Purpose |
|-------|-------|---------|
| `hostname` | `web` | Subdomain for the A record |
| `domain` | `example.com` | Route53 hosted zone |
| `ttl` | `300` | DNS record TTL in seconds |
| `zone_id` | `""` | Route53 zone ID (empty = look up from domain) |
| `is_apex_domain` | `false` | Set `true` to create an A record at the zone apex |
| `cname_aliases` | `["www"]` | Additional CNAME records pointing at the hostname |

With the config above and `is_apex_domain: false`, goloo creates:
- `web.example.com` → A record → instance IP
- `www.example.com` → CNAME → `web.example.com`

With `is_apex_domain: true`, goloo also creates:
- `example.com` → A record → instance IP

### Supported OS values

`ubuntu-24.04`, `ubuntu-22.04`, `ubuntu-20.04`, `amazon-linux-2023`, `amazon-linux-2`, `debian-12`, `debian-11`

### Runtime fields (populated by goloo)

These fields are written to the config after creation — you don't set them:

`stack_id`, `stack_name`, `ami_id`, `instance_id`, `public_ip`, `security_group`, `zone_id` (if looked up), `fqdn`, `dns_records`

## Before you start

1. AWS credentials configured: `aws configure`
2. A Route53 hosted zone for your domain
3. Build goloo: `make build` from the project root
4. Edit `config.json`:
   - Replace `your-github-username` with your GitHub username
   - Replace `your-deploy-bot-username` with a second GitHub account (or remove the second user)
   - Replace `example.com` with your Route53 hosted zone domain
   - Set `vpc_id` and `subnet_id` if you have a specific VPC (or leave empty for auto-discovery)
   - Set `zone_id` if you know it (or leave empty for auto-lookup from domain)
   - Set `is_apex_domain` to `true` if this server should also serve the bare domain

## Deploy to AWS

From the project root:

```bash
mkdir -p stacks/aws-web-server
cp examples/aws-web-server/config.json examples/aws-web-server/cloud-init.yaml stacks/aws-web-server/

goloo create aws-web-server --aws
```

Or use the `-u` flag to skip editing users in the config:

```bash
goloo create aws-web-server --aws -f ./examples/ -u gherlein
```

## Verify it worked

```bash
goloo status aws-web-server
```

SSH in and check nginx:

```bash
goloo ssh aws-web-server
systemctl status nginx
curl http://localhost
```

Or from your local machine using the DNS name:

```bash
curl http://web.example.com
```

## Set up TLS

Once DNS is resolving, SSH in and run certbot:

```bash
goloo ssh aws-web-server
sudo certbot --nginx -d web.example.com
```

If using apex domain and www alias:

```bash
sudo certbot --nginx -d example.com -d www.example.com -d web.example.com
```

## Blue-green deployment

Deploy a new version alongside the old one, then swap DNS:

```bash
goloo create aws-web-server-v2 --aws
# test the new server...
goloo dns swap aws-web-server-v2
# web.example.com now points at the new server
goloo delete aws-web-server
```

## Stop and clean up

```bash
goloo stop aws-web-server
goloo start aws-web-server

goloo delete aws-web-server
```

## Files

```
aws-web-server/
├── README.md
├── config.json            # Full AWS config: instance, network, DNS, users
└── cloud-init.yaml        # cloud-init: nginx, certbot, fail2ban, ufw, tools
```
