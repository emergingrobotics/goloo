# Goloo Design Document

Unified CLI tool for provisioning VMs locally (Multipass) or in the cloud (AWS) using the same configuration and cloud-init files.

## Problem Statement

Development teams need consistent environments across local machines and cloud infrastructure. Currently:

1. **Local VMs** use Multipass with cloud-init YAML
2. **Cloud VMs** use AWS CLI/Console with the same cloud-init YAML
3. **No unified CLI** bridges these environments with identical configs

The cloud-init format is already portable. The missing piece is a single tool that abstracts provider differences while keeping configs identical.

## Solution

Goloo provides a unified interface:

```bash
goloo create devbox              # Local VM (Multipass)
goloo create devbox --aws        # AWS EC2 instance
```

Same config file, same cloud-init, different targets.

## Core Principle

**One config, any target.** A goloo config file plus a cloud-init YAML should:
1. Create a local VM via `goloo create myvm`
2. Create an identical AWS EC2 instance via `goloo create myvm --aws`

The developer workflow is: develop locally, deploy to cloud with a flag change.

---

## System Architecture

### Block Diagram

```mermaid
flowchart TB
    subgraph CLI["CLI Layer"]
        CMD[Command Parser]
        CFG[Config Loader]
    end

    subgraph Core["Core Layer"]
        DISP[Provider Dispatcher]
        CINIT[Cloud-Init Processor]
        SSH[SSH Key Fetcher]
    end

    subgraph Providers["Provider Layer"]
        MP[Multipass Provider]
        AWS[AWS Provider]
    end

    subgraph MultipassExt["Multipass External"]
        MPBIN[multipass CLI]
    end

    subgraph AWSExt["AWS External"]
        CFN[CloudFormation]
        R53[Route53]
        SSM[SSM Parameter Store]
        EC2[EC2]
    end

    subgraph Storage["Storage"]
        STACKS[(stacks/*.json)]
        CLOUDINIT[(cloud-init/*.yaml)]
    end

    subgraph External["External APIs"]
        GH[GitHub API]
    end

    CMD --> CFG
    CFG --> STACKS
    CFG --> DISP
    DISP --> CINIT
    CINIT --> CLOUDINIT
    CINIT --> SSH
    SSH --> GH

    DISP --> MP
    DISP --> AWS

    MP --> MPBIN

    AWS --> CFN
    AWS --> R53
    AWS --> SSM
    AWS --> EC2
```

### Component Responsibilities

| Component | Responsibility |
|-----------|----------------|
| Command Parser | Parse CLI flags (`create`, `delete`, `--aws`, etc.) |
| Config Loader | Read JSON configs from `stacks/`, apply defaults, validate |
| Provider Dispatcher | Route to Multipass or AWS based on flags/config |
| Cloud-Init Processor | Substitute `${SSH_PUBLIC_KEY}` variables, validate YAML |
| SSH Key Fetcher | Fetch public keys from `https://github.com/{user}.keys` |
| Multipass Provider | Shell out to `multipass` CLI, parse JSON output |
| AWS Provider | Use AWS SDK for CloudFormation, Route53, SSM, EC2 |

---

## How It Works

### Local VMs (Multipass)

Goloo shells out to `multipass`. The config file translates directly to CLI parameters:

```
goloo config          →    multipass command
─────────────────────────────────────────────
vm.name               →    --name
vm.cpus               →    --cpus
vm.memory             →    --memory
vm.disk               →    --disk
vm.image              →    (positional arg)
vm.cloud_init_file    →    --cloud-init
```

The cloud-init YAML passes through unchanged after SSH key substitution.

### AWS VMs

Goloo uses the AWS Go SDK (based on [aws-ec2](https://github.com/emergingrobotics/aws-ec2)) to:

1. Look up AMI ID via SSM Parameter Store
2. Discover or create VPC/Subnet infrastructure
3. Create CloudFormation stack (EC2 instance + security group)
4. Pass cloud-init YAML as EC2 UserData (base64 encoded)
5. Create Route53 DNS record: `{hostname}.{domain}` → instance IP
6. Update config file with output fields (instance_id, public_ip, etc.)

---

## Data Flow

### VM Creation Sequence (Local)

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Config
    participant CloudInit
    participant GitHub
    participant Multipass

    User->>CLI: goloo create devbox
    CLI->>Config: Load stacks/devbox.json
    Config-->>CLI: VMConfig

    CLI->>CloudInit: Process(cloud-init/dev.yaml, users)
    CloudInit->>GitHub: GET /gherlein.keys
    GitHub-->>CloudInit: SSH public keys
    CloudInit->>CloudInit: Substitute ${SSH_PUBLIC_KEY}
    CloudInit-->>CLI: Processed YAML path

    CLI->>Multipass: multipass launch 24.04 --name devbox --cpus 4 --memory 4G --disk 40G --cloud-init /tmp/processed.yaml
    Multipass-->>CLI: VM created

    CLI->>Multipass: multipass info devbox --format json
    Multipass-->>CLI: {"info":{"devbox":{"ipv4":["192.168.64.5"]}}}

    CLI->>Config: Update config with IP
    CLI-->>User: VM created at 192.168.64.5
```

### VM Creation Sequence (AWS)

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Config
    participant CloudInit
    participant GitHub
    participant SSM
    participant EC2
    participant CFN
    participant R53

    User->>CLI: goloo create devbox --aws
    CLI->>Config: Load stacks/devbox.json
    Config-->>CLI: VMConfig + DNSConfig

    CLI->>CloudInit: Process(cloud-init/dev.yaml, users)
    CloudInit->>GitHub: GET /gherlein.keys
    GitHub-->>CloudInit: SSH public keys
    CloudInit-->>CLI: Processed cloud-init content

    CLI->>SSM: GetParameter(/aws/service/canonical/ubuntu/...)
    SSM-->>CLI: ami-0123456789

    CLI->>EC2: DescribeVpcs (find default VPC)
    EC2-->>CLI: vpc-abc123
    CLI->>EC2: DescribeSubnets
    EC2-->>CLI: subnet-def456

    CLI->>CFN: CreateStack(template, params, UserData)
    CFN-->>CLI: StackId

    CLI->>CFN: Wait for CREATE_COMPLETE
    CFN-->>CLI: Success

    CLI->>CFN: DescribeStacks (get outputs)
    CFN-->>CLI: InstanceId, PublicIP, SecurityGroupId

    CLI->>R53: ListHostedZonesByName(domain)
    R53-->>CLI: ZoneId
    CLI->>R53: ChangeResourceRecordSets (A record)
    R53-->>CLI: Success

    CLI->>Config: Update config with outputs
    CLI-->>User: VM created at devbox.example.com (54.1.2.3)
```

### VM Deletion Sequence

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Config
    participant Provider
    participant External

    User->>CLI: goloo delete devbox
    CLI->>Config: Load stacks/devbox.json
    Config-->>CLI: Config (with provider detection)

    alt Multipass (no stack_id)
        CLI->>Provider: Delete via multipass
        Provider->>External: multipass delete devbox
        External-->>Provider: OK
        Provider->>External: multipass purge
        External-->>Provider: OK
    else AWS (has stack_id)
        CLI->>Provider: Delete via AWS
        Provider->>External: Route53 DELETE records
        External-->>Provider: OK
        Provider->>External: CloudFormation DeleteStack
        External-->>Provider: Deletion initiated
        Provider->>External: Wait for DELETE_COMPLETE
        External-->>Provider: OK
        Provider->>External: Delete created VPC/Subnet (if any)
        External-->>Provider: OK
    end

    CLI->>Config: Clear output fields
    CLI-->>User: VM deleted
```

---

## Configuration Schema

### Config File Structure

```mermaid
erDiagram
    Config ||--o| VMConfig : contains
    Config ||--o| DNSConfig : contains
    VMConfig ||--|{ User : has
    VMConfig ||--o{ Mount : has

    Config {
        string provider "optional: multipass or aws"
    }

    VMConfig {
        string name
        int cpus
        string memory
        string disk
        string image
        string cloud_init_file
        string instance_type "AWS only"
        string region "AWS only"
        string vpc_id "AWS only, auto-discovered"
        string subnet_id "AWS only, auto-discovered"
        string public_ip "output"
        string instance_id "output"
        string stack_id "output, AWS only"
    }

    User {
        string username
        string github_username
    }

    Mount {
        string source "Multipass only"
        string target "Multipass only"
    }

    DNSConfig {
        string hostname
        string domain
        int ttl
        bool is_apex_domain
        string[] cname_aliases
        string zone_id "output"
        string fqdn "output"
    }
```

### Example Configs

**Local Development (Multipass):**

```json
{
  "vm": {
    "name": "devbox",
    "cpus": 4,
    "memory": "4G",
    "disk": "40G",
    "image": "24.04",
    "cloud_init_file": "cloud-init/dev.yaml",
    "users": [
      {"username": "ubuntu", "github_username": "gherlein"}
    ],
    "mounts": [
      {"source": "/Users/gherlein/src", "target": "/home/ubuntu/src"}
    ]
  }
}
```

**Cloud Development (AWS):**

```json
{
  "vm": {
    "name": "devbox",
    "instance_type": "t3.medium",
    "os": "ubuntu-24.04",
    "region": "us-west-2",
    "cloud_init_file": "cloud-init/dev.yaml",
    "users": [
      {"username": "ubuntu", "github_username": "gherlein"}
    ]
  },
  "dns": {
    "hostname": "devbox",
    "domain": "example.com",
    "ttl": 300
  }
}
```

**Same Config, Both Targets:**

A single config can work for both (DNS ignored for local):

```json
{
  "vm": {
    "name": "devbox",
    "cpus": 4,
    "memory": "4G",
    "disk": "40G",
    "image": "24.04",
    "instance_type": "t3.medium",
    "os": "ubuntu-24.04",
    "region": "us-west-2",
    "cloud_init_file": "cloud-init/dev.yaml",
    "users": [
      {"username": "ubuntu", "github_username": "gherlein"}
    ]
  },
  "dns": {
    "hostname": "devbox",
    "domain": "example.com",
    "ttl": 300
  }
}
```

---

## Provider Interface

```go
package provider

import (
    "context"
    "time"
)

type VMProvider interface {
    Name() string
    Create(ctx context.Context, cfg *Config, cloudInitPath string) error
    Delete(ctx context.Context, cfg *Config) error
    Status(ctx context.Context, cfg *Config) (*VMStatus, error)
    List(ctx context.Context) ([]VMStatus, error)
    SSH(ctx context.Context, cfg *Config) error
    Stop(ctx context.Context, cfg *Config) error
    Start(ctx context.Context, cfg *Config) error
}

type VMStatus struct {
    Name      string
    State     string    // Running, Stopped, etc.
    IP        string
    Provider  string    // multipass, aws
    CreatedAt time.Time
}
```

### Provider Class Diagram

```mermaid
classDiagram
    class VMProvider {
        <<interface>>
        +Name() string
        +Create(ctx, cfg, cloudInitPath) error
        +Delete(ctx, cfg) error
        +Status(ctx, cfg) VMStatus
        +List(ctx) []VMStatus
        +SSH(ctx, cfg) error
        +Stop(ctx, cfg) error
        +Start(ctx, cfg) error
    }

    class MultipassProvider {
        +Name() string
        +Create(ctx, cfg, cloudInitPath) error
        +Delete(ctx, cfg) error
        +Status(ctx, cfg) VMStatus
        +List(ctx) []VMStatus
        +SSH(ctx, cfg) error
        +Stop(ctx, cfg) error
        +Start(ctx, cfg) error
        -runCommand(args) []byte, error
        -parseInfo(json) MultipassInfo
    }

    class AWSProvider {
        -cfnClient CloudFormationClient
        -ec2Client EC2Client
        -r53Client Route53Client
        -ssmClient SSMClient
        +Name() string
        +Create(ctx, cfg, cloudInitPath) error
        +Delete(ctx, cfg) error
        +Status(ctx, cfg) VMStatus
        +List(ctx) []VMStatus
        +SSH(ctx, cfg) error
        +Stop(ctx, cfg) error
        +Start(ctx, cfg) error
        -lookupAMI(os) string
        -discoverVPC() string
        -createNetworkStack() NetworkStack
        -generateCFNTemplate(userData) string
    }

    VMProvider <|.. MultipassProvider
    VMProvider <|.. AWSProvider
```

---

## Cloud-Init Processing

### Variable Substitution Flow

```mermaid
flowchart LR
    A[cloud-init/dev.yaml] --> B[Read Template]
    B --> C{Has Users?}
    C -->|Yes| D[Fetch GitHub Keys]
    D --> E[github.com/user.keys]
    E --> F[Substitute Variables]
    C -->|No| F
    F --> G[Write Temp File]
    G --> H[Return Path]

    subgraph Variables
        V1["${SSH_PUBLIC_KEY}"]
        V2["${SSH_PUBLIC_KEY_USERNAME}"]
    end
```

### Supported Variables

| Variable | Source | Description |
|----------|--------|-------------|
| `${SSH_PUBLIC_KEY}` | GitHub | First user's SSH public keys |
| `${SSH_PUBLIC_KEY_<USERNAME>}` | GitHub | Specific user's SSH keys (uppercase) |

### Example Cloud-Init

```yaml
#cloud-config
users:
  - name: ubuntu
    groups: sudo
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh_authorized_keys:
      - ${SSH_PUBLIC_KEY}

packages:
  - git
  - build-essential
  - golang-go

runcmd:
  - echo "Cloud-init complete" > /var/log/goloo-init.log
```

---

## AWS Implementation Details

### CloudFormation Template

Based on [aws-ec2](https://github.com/emergingrobotics/aws-ec2), the generated template includes:

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Description: EC2 instance with SSH access

Parameters:
  ImageId:
    Type: String
  InstanceType:
    Type: String
    Default: t3.micro
  VpcId:
    Type: String
  SubnetId:
    Type: String

Resources:
  SSHSecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: Allow SSH/HTTP/HTTPS
      VpcId: !Ref VpcId
      SecurityGroupIngress:
        - IpProtocol: tcp
          FromPort: 22
          ToPort: 22
          CidrIp: 0.0.0.0/0
        - IpProtocol: tcp
          FromPort: 80
          ToPort: 80
          CidrIp: 0.0.0.0/0
        - IpProtocol: tcp
          FromPort: 443
          ToPort: 443
          CidrIp: 0.0.0.0/0

  EC2Instance:
    Type: AWS::EC2::Instance
    Properties:
      InstanceType: !Ref InstanceType
      ImageId: !Ref ImageId
      NetworkInterfaces:
        - DeviceIndex: "0"
          SubnetId: !Ref SubnetId
          AssociatePublicIpAddress: true
          GroupSet:
            - !GetAtt SSHSecurityGroup.GroupId
      UserData: <base64-encoded-cloud-init>

Outputs:
  InstanceId:
    Value: !Ref EC2Instance
  PublicIP:
    Value: !GetAtt EC2Instance.PublicIp
  SecurityGroupId:
    Value: !Ref SSHSecurityGroup
```

### AMI Lookup via SSM

```go
var osSSMPaths = map[string]string{
    "ubuntu-24.04": "/aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp2/ami-id",
    "ubuntu-22.04": "/aws/service/canonical/ubuntu/server/22.04/stable/current/amd64/hvm/ebs-gp2/ami-id",
    "amazon-linux-2023": "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64",
}
```

### VPC/Subnet Discovery

1. Try to find default VPC
2. If no default VPC, find any available VPC
3. If no VPC exists, create network stack (VPC + Subnet + IGW + Route Table)
4. Find subnet with public IP auto-assign, or create one

### DNS Record Creation

1. Look up hosted zone ID for domain
2. Create A record: `hostname.domain` → public IP
3. Optionally create CNAME aliases
4. Optionally create apex domain record

---

## CLI Design

### Command Structure

```
goloo <command> <name> [flags]

Commands:
  create    Create a new VM
  delete    Delete a VM
  status    Show VM status
  list      List all VMs
  ssh       SSH into a VM
  stop      Stop a VM
  start     Start a VM

Flags:
  --aws         Use AWS provider (override config)
  --local       Use Multipass provider (override config)
  --config      Path to config file
  --cloud-init  Override cloud-init file

Backwards Compatibility (aws-ec2 style):
  -c            Shorthand for create
  -d            Shorthand for delete
  -n            Shorthand for name
```

### Provider Selection Logic

```mermaid
flowchart TD
    A[Parse Flags] --> B{--local flag?}
    B -->|Yes| C[Use Multipass]
    B -->|No| D{--aws flag?}
    D -->|Yes| E[Use AWS]
    D -->|No| F[Load Config]
    F --> G{Has stack_id?}
    G -->|Yes| H[Use AWS - existing stack]
    G -->|No| I{Has dns.domain?}
    I -->|Yes| J[Default: AWS]
    I -->|No| K[Default: Multipass]
```

---

## State Management

### Config File Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Input: User creates config

    Input --> LocalRunning: goloo create (local)
    Input --> AWSRunning: goloo create --aws

    LocalRunning --> LocalRunning: Fields populated
    note right of LocalRunning
        vm.public_ip set
    end note

    AWSRunning --> AWSRunning: Fields populated
    note right of AWSRunning
        vm.instance_id
        vm.public_ip
        vm.stack_id
        dns.zone_id
        dns.fqdn
    end note

    LocalRunning --> LocalStopped: goloo stop
    AWSRunning --> AWSStopped: goloo stop

    LocalStopped --> LocalRunning: goloo start
    AWSStopped --> AWSRunning: goloo start

    LocalRunning --> Input: goloo delete
    LocalStopped --> Input: goloo delete
    AWSRunning --> Input: goloo delete
    AWSStopped --> Input: goloo delete

    Input --> [*]: User deletes file
```

### Output Fields by Provider

| Field | Multipass | AWS |
|-------|-----------|-----|
| `vm.instance_id` | - | EC2 instance ID |
| `vm.public_ip` | Multipass IP | EC2 public IP |
| `vm.stack_id` | - | CloudFormation stack ARN |
| `vm.stack_name` | - | CloudFormation stack name |
| `vm.security_group` | - | Security group ID |
| `vm.ami_id` | - | AMI ID used |
| `vm.vpc_id` | - | VPC ID (discovered/created) |
| `vm.subnet_id` | - | Subnet ID (discovered/created) |
| `dns.zone_id` | - | Route53 zone ID |
| `dns.fqdn` | - | Full DNS name |
| `dns.dns_records` | - | Created DNS records |

---

## Update Strategy: CNAME Swap

For zero-downtime updates, goloo supports blue-green deployment via DNS:

```mermaid
sequenceDiagram
    participant User
    participant DNS
    participant Blue as VM (Blue v1.0)
    participant Green as VM (Green v1.1)

    Note over Blue: Running v1.0
    User->>DNS: app.example.com
    DNS->>Blue: 10.0.1.100

    User->>Green: goloo create app-v2 --aws
    Note over Green: Deploy v1.1
    Green->>Green: Cloud-init runs

    User->>Green: Test green VM

    User->>DNS: goloo dns swap app
    Note over DNS: Update A record
    DNS->>DNS: app.example.com → 10.0.2.200

    User->>DNS: app.example.com
    DNS->>Green: 10.0.2.200

    Note over Blue: Keep 1hr for rollback
    User->>Blue: goloo delete app-v1
```

---

## Directory Structure

```
goloo/
├── cmd/goloo/
│   └── main.go              # CLI entry point
├── internal/
│   ├── config/
│   │   ├── config.go        # Config types
│   │   ├── loader.go        # Load, save, validate
│   │   └── loader_test.go
│   ├── cloudinit/
│   │   ├── processor.go     # Variable substitution
│   │   ├── ssh.go           # GitHub key fetching
│   │   └── processor_test.go
│   └── provider/
│       ├── interface.go     # VMProvider interface
│       ├── registry.go      # Provider registry
│       ├── multipass/
│       │   ├── multipass.go # Shell out to multipass
│       │   └── multipass_test.go
│       └── aws/
│           ├── aws.go       # Main provider
│           ├── template.go  # CloudFormation template
│           ├── dns.go       # Route53 operations
│           ├── network.go   # VPC/Subnet discovery
│           └── aws_test.go
├── stacks/                  # Config files (gitignored)
│   └── devbox.json
├── cloud-init/              # Cloud-init templates
│   ├── base.yaml
│   ├── dev.yaml
│   └── claude-dev.yaml
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

---

## Error Handling

### Error Categories

```mermaid
flowchart TD
    E[Error] --> V[Validation Error]
    E --> P[Provider Error]
    E --> N[Network Error]
    E --> C[Config Error]

    V --> V1[Missing required field]
    V --> V2[Invalid username format]
    V --> V3[Cloud-init syntax error]

    P --> P1[Multipass not installed]
    P --> P2[AWS credentials missing]
    P --> P3[Resource already exists]
    P --> P4[Resource not found]
    P --> P5[VPC/Subnet not found]

    N --> N1[GitHub unreachable]
    N --> N2[AWS API timeout]
    N --> N3[CloudFormation failed]

    C --> C1[Config file not found]
    C --> C2[Invalid JSON]
    C --> C3[Unsupported OS]
```

### Error Messages

All errors should be actionable:

```go
var (
    ErrConfigNotFound   = errors.New("config file not found: create stacks/<name>.json")
    ErrProviderUnknown  = errors.New("unknown provider: use 'multipass' or 'aws'")
    ErrVMExists         = errors.New("VM already exists: delete first with 'goloo delete <name>'")
    ErrVMNotFound       = errors.New("VM not found: check 'goloo list' for available VMs")
    ErrNoSSHKeys        = errors.New("no SSH keys found: verify GitHub username at github.com/<user>.keys")
    ErrNoVPC            = errors.New("no VPC found: will create new network infrastructure")
    ErrAWSCredentials   = errors.New("AWS credentials not configured: run 'aws configure'")
    ErrMultipassMissing = errors.New("multipass not installed: visit multipass.run")
)
```

---

## Security Considerations

1. **SSH Keys**: Fetched from GitHub over HTTPS only
2. **AWS Credentials**: Use standard AWS credential chain (env, ~/.aws, IAM role)
3. **Config Files**: May contain sensitive data - gitignored by default
4. **Cloud-Init**: Runs as root - review templates carefully
5. **Security Groups**: SSH open to 0.0.0.0/0 by default - restrict for production
6. **DNS TTL**: Set low (60-300s) to enable quick CNAME swaps

---

## Dependencies

### Go Modules

```go
require (
    github.com/aws/aws-sdk-go-v2 v1.x
    github.com/aws/aws-sdk-go-v2/config v1.x
    github.com/aws/aws-sdk-go-v2/service/cloudformation v1.x
    github.com/aws/aws-sdk-go-v2/service/ec2 v1.x
    github.com/aws/aws-sdk-go-v2/service/route53 v1.x
    github.com/aws/aws-sdk-go-v2/service/ssm v1.x
)
```

### External Dependencies

| Dependency | Required For |
|------------|--------------|
| `multipass` CLI | Local VMs |
| AWS credentials | AWS provider |
| Internet access | GitHub SSH keys, AWS API |

---

## References

- [Multipass Documentation](https://multipass.run/)
- [cloud-init Documentation](https://cloudinit.readthedocs.io/)
- [AWS CloudFormation](https://docs.aws.amazon.com/cloudformation/)
- [aws-ec2 Repository](https://github.com/emergingrobotics/aws-ec2) - Foundation for AWS provider
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/)
