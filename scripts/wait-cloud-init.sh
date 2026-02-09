#!/bin/bash
# Wait for cloud-init to complete on a VM

set -e

NAME="${1:-}"
TIMEOUT="${2:-600}"

if [ -z "$NAME" ]; then
    echo "Usage: $0 VM_NAME [TIMEOUT_SECONDS]"
    echo ""
    echo "Waits for cloud-init to complete on the specified VM"
    echo "Default timeout: 600 seconds"
    exit 1
fi

echo "Waiting for cloud-init to complete on '$NAME' (timeout: ${TIMEOUT}s)..."

if ! multipass exec "$NAME" -- cloud-init status --wait --long 2>/dev/null; then
    echo ""
    echo "Cloud-init may have encountered errors. Check logs with:"
    echo "  multipass exec $NAME -- cat /var/log/cloud-init-output.log"
    exit 1
fi

echo ""
echo "Cloud-init completed successfully!"
