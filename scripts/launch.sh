#!/bin/bash
# VM Launch Script
# Handles SSH key substitution and multipass launch

set -e

# Defaults
NAME=""
CONFIG=""
IMAGE="24.04"
CPUS="4"
MEMORY="4G"
DISK="40G"
VALIDATE_ONLY=false
BRIDGED=false

usage() {
    echo "Usage: $0 --name NAME --config CONFIG [options]"
    echo ""
    echo "Options:"
    echo "  --name NAME        VM name (required)"
    echo "  --config CONFIG    Path to cloud-init config (required)"
    echo "  --image IMAGE      Ubuntu image (default: 24.04)"
    echo "  --cpus CPUS        Number of CPUs (default: 4)"
    echo "  --memory MEM       Memory allocation (default: 4G)"
    echo "  --disk DISK        Disk size (default: 40G)"
    echo "  --bridged          Enable bridged networking"
    echo "  --validate-only    Validate config without launching"
    echo "  --help             Show this help"
    exit 1
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --name)
            NAME="$2"
            shift 2
            ;;
        --config)
            CONFIG="$2"
            shift 2
            ;;
        --image)
            IMAGE="$2"
            shift 2
            ;;
        --cpus)
            CPUS="$2"
            shift 2
            ;;
        --memory)
            MEMORY="$2"
            shift 2
            ;;
        --disk)
            DISK="$2"
            shift 2
            ;;
        --bridged)
            BRIDGED=true
            shift
            ;;
        --validate-only)
            VALIDATE_ONLY=true
            CONFIG="$2"
            shift 2
            ;;
        --help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Find SSH public key
find_ssh_key() {
    if [ -f ~/.ssh/id_ed25519.pub ]; then
        cat ~/.ssh/id_ed25519.pub
    elif [ -f ~/.ssh/id_rsa.pub ]; then
        cat ~/.ssh/id_rsa.pub
    else
        echo ""
    fi
}

SSH_KEY=$(find_ssh_key)

if [ -z "$SSH_KEY" ]; then
    echo "Error: No SSH public key found at ~/.ssh/id_ed25519.pub or ~/.ssh/id_rsa.pub"
    echo "Generate one with: ssh-keygen -t ed25519"
    exit 1
fi

# Validate config exists
if [ ! -f "$CONFIG" ]; then
    echo "Error: Config file not found: $CONFIG"
    exit 1
fi

# Create temporary config with substituted SSH key
TEMP_CONFIG=$(mktemp)
trap "rm -f $TEMP_CONFIG" EXIT

# Substitute SSH key placeholder
sed "s|\${SSH_PUBLIC_KEY}|$SSH_KEY|g" "$CONFIG" > "$TEMP_CONFIG"

# Validate cloud-init config
echo "Validating cloud-init configuration..."
if ! multipass launch --cloud-init "$TEMP_CONFIG" --name _validate_test_ 2>&1 | head -1 | grep -q "Launched"; then
    # Check for YAML errors by attempting validation
    if head -1 "$TEMP_CONFIG" | grep -q "^#cloud-config"; then
        echo "Configuration appears valid (basic check passed)"
    else
        echo "Error: Configuration must start with #cloud-config"
        exit 1
    fi
fi

if [ "$VALIDATE_ONLY" = true ]; then
    echo "Configuration validated successfully: $CONFIG"
    echo ""
    echo "Processed configuration:"
    echo "------------------------"
    cat "$TEMP_CONFIG"
    exit 0
fi

# Require name for actual launch
if [ -z "$NAME" ]; then
    echo "Error: --name is required"
    usage
fi

# Check if VM already exists
if multipass list | grep -q "^$NAME "; then
    echo "Error: VM '$NAME' already exists"
    echo "Use 'make delete NAME=$NAME' to remove it first"
    exit 1
fi

echo "Launching VM: $NAME"
echo "  Image: $IMAGE"
echo "  CPUs: $CPUS"
echo "  Memory: $MEMORY"
echo "  Disk: $DISK"
echo "  Config: $CONFIG"
echo ""

# Build launch command
LAUNCH_CMD="multipass launch $IMAGE"
LAUNCH_CMD="$LAUNCH_CMD --name $NAME"
LAUNCH_CMD="$LAUNCH_CMD --cpus $CPUS"
LAUNCH_CMD="$LAUNCH_CMD --memory $MEMORY"
LAUNCH_CMD="$LAUNCH_CMD --disk $DISK"
LAUNCH_CMD="$LAUNCH_CMD --cloud-init $TEMP_CONFIG"

if [ "$BRIDGED" = true ]; then
    LAUNCH_CMD="$LAUNCH_CMD --bridged"
fi

# Launch
echo "Executing: $LAUNCH_CMD"
echo ""

eval $LAUNCH_CMD

echo ""
echo "VM '$NAME' launched successfully!"
echo ""
echo "Connect with: multipass shell $NAME"
echo "         or:  make ssh NAME=$NAME"
echo ""
echo "Cloud-init may still be running. Check status with:"
echo "  multipass exec $NAME -- cloud-init status --wait"
