#!/bin/sh
set -e

if command -v systemctl >/dev/null 2>&1; then
    systemctl stop secban-mailer.service 2>/dev/null || true
    systemctl disable secban-mailer.service 2>/dev/null || true
fi
