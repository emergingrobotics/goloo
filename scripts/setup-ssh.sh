#!/bin/bash
# Setup SSH config for easy VM access
# Adds entries to ~/.ssh/config for multipass VMs

set -e

NAME="${1:-}"

if [ -z "$NAME" ]; then
    echo "Usage: $0 VM_NAME"
    echo ""
    echo "Adds SSH config entry for the specified Multipass VM"
    exit 1
fi

# Get VM IP
IP=$(multipass info "$NAME" --format csv | tail -1 | cut -d',' -f3)

if [ -z "$IP" ] || [ "$IP" = "N/A" ]; then
    echo "Error: Could not get IP for VM '$NAME'"
    echo "Is the VM running? Check with: multipass list"
    exit 1
fi

# Check if entry already exists
if grep -q "^Host $NAME$" ~/.ssh/config 2>/dev/null; then
    echo "SSH config entry for '$NAME' already exists"
    echo "Current IP in multipass: $IP"
    echo ""
    echo "To update, remove the existing entry from ~/.ssh/config and run again"
    exit 0
fi

# Create backup
if [ -f ~/.ssh/config ]; then
    cp ~/.ssh/config ~/.ssh/config.bak
fi

# Add entry
cat >> ~/.ssh/config << EOF

# Multipass VM: $NAME
Host $NAME
    HostName $IP
    User ubuntu
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    LogLevel ERROR
EOF

echo "Added SSH config for '$NAME' at $IP"
echo ""
echo "Connect with: ssh $NAME"
echo ""
echo "Note: IP may change if VM is restarted. Run this script again to update."
