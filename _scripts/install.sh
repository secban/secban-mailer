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

# Create user for secban in the secban group.
if ! id -u secban >/dev/null 2>&1; then
    useradd --no-create-home --system --shell /sbin/nologin --gid secban secban
fi

# Install the secban-mailer binary.
cp "./secban-mailer" "/usr/sbin/"
chmod +x "/usr/sbin/secban-mailer"

# Install the emailcheck binary.
cp "./secban-mailer-emailcheck" "/usr/bin/"
chmod +x "/usr/bin/secban-mailer-emailcheck"

# Install the example mailer configuration.
mkdir -p /etc/secban
if [ ! -f /etc/secban/mailer.yaml ]; then
    cp "./mailer.yaml" "/etc/secban/mailer.yaml"
fi

# Install the systemd service file for the secban-mailer.
if [ ! -f /etc/systemd/system/secban-mailer.service ]; then
    cp "./secban-mailer.service" "/etc/systemd/system/secban-mailer.service"
fi

if [ ! -z "$(which systemctl)" ]; then
    systemctl daemon-reload
    systemctl enable secban-mailer.service
fi