#!/bin/bash
# Script to configure /etc/hosts for local Kubernetes development

HOSTS_ENTRY="127.0.0.1 firstclick.local"

# Check if the entry already exists
if grep -q "$HOSTS_ENTRY" /etc/hosts; then
    echo "✓ /etc/hosts already configured with: $HOSTS_ENTRY"
    exit 0
fi

# Add the entry to /etc/hosts
echo "Adding entry to /etc/hosts..."
echo "$HOSTS_ENTRY" | sudo tee -a /etc/hosts > /dev/null

if grep -q "$HOSTS_ENTRY" /etc/hosts; then
    echo "✓ Successfully added to /etc/hosts"
    echo "✓ You can now access: http://firstclick.local"
else
    echo "✗ Failed to add entry to /etc/hosts"
    exit 1
fi
