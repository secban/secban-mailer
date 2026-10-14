#!/bin/bash
set -ex

OWNER=$(whoami)
if [ "$OWNER" != "root" ]; then
    echo "This script must be run as root"
    exit 1
fi

# Ensure our location.
OWN_DIR=$(dirname "$0")
cd "$OWN_DIR"

# Create secban group if not exists.
if ! getent group secban >/dev/null 2>&1; then
    groupadd --system secban
fi

# Create user for the secban-agent in the secban group.
if ! id -u secban-mailer >/dev/null 2>&1; then
    useradd --no-create-home --system --shell /usr/sbin/nologin --gid secban secban-mailer
fi

# Install the secban-mailer binary.
cp "./secban-mailer" "/usr/sbin/"
chmod +x "/usr/sbin/secban-mailer"

# Install the example mailer configuration.
mkdir -p /etc/secban
if [ ! -f /etc/secban/mailer.yaml ]; then
    cp "./mailer.yaml" "/etc/secban/mailer.yaml"
fi

# Install the systemd service file for the secban-mailer.
cp "./secban-mailer.service" "/etc/systemd/system/secban-mailer.service"
systemctl daemon-reload