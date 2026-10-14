#!/bin/bash
set -ex

REPO=$(git rev-parse --show-toplevel)

rm -rf build
mkdir -p build

BUILD_OPTIONS='-trimpath'
LDFLAGS="-ldflags=-w -s"

go build "$BUILD_OPTIONS" -o "$REPO/build/secban-mailer" "$LDFLAGS" "$REPO/cmd/mailer/mailer.go"
go build "$BUILD_OPTIONS" -o "$REPO/build/secban-mailer-emailcheck" "$LDFLAGS" "$REPO/cmd/emailcheck/emailcheck.go"

cp "$REPO/scripts/install.sh" "$REPO/build/"
cp "$REPO/config/mailer.yaml" "$REPO/build/"
cp "$REPO/scripts/systemd/secban-mailer.service" "$REPO/build/"

chmod +x "$REPO/build/install.sh"