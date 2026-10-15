# SecBan Mailer

SecBan Mailer is an alert dispatch daemon for the SecBan framework. It listens on a UNIX domain socket for IPC notification messages sent by the SecBan agent upon IP ban events and delivers formatted notification emails via SMTP.

## Features

- IPC receiver via UNIX domain socket
- SMTP email dispatch with authentication support (auto-discovery)
- Configurable TLS policy (`notsl`, `opportunistic`, `mandatory`)
- Environment variable overrides for sensitive credentials (`SECBAN_MAILER_SMTP_USERNAME`, `SECBAN_MAILER_SMTP_PASSWORD`)
- Customizable plain text email body via Go templates
- CLI diagnostic tool (`secban-mailer-emailcheck`) to verify SMTP configuration and send test alerts
- Systemd service integration

## Installation

Pre-compiled release packages are published in the GitLab package registry.

### Download and Extract

```sh
# Set target version, OS, and architecture
VERSION="v0.1.0" # replace with desired version
OS="linux"
ARCH="amd64" # or arm64

# Download archive from GitLab Generic Package Registry
curl -sSL -o "secban-mailer.tar.gz" \
  "https://gitlab.com/api/v4/projects/secban%2Fsecban-mailer/packages/generic/secban-mailer/${VERSION}/secban-mailer-${OS}-${ARCH}-${VERSION}.tar.gz"

# Extract archive
mkdir -p /tmp/secban-install
tar -xzf "secban-mailer.tar.gz" -C /tmp/secban-install
cd /tmp/secban-install
```

### Install

Run the included install script as `root`:

```sh
sudo ./install.sh
```

The script performs the following steps:
1. Creates `secban` system group and `secban-mailer` system user.
2. Installs daemon `/usr/sbin/secban-mailer`.
3. Installs test utility `/usr/bin/secban-mailer-emailcheck`.
4. Copies default configuration to `/etc/secban/mailer.yaml` (if not already present).
5. Installs and enables systemd service `secban-mailer.service`.

### Start Service

```sh
sudo systemctl start secban-mailer.service
sudo systemctl status secban-mailer.service
```

## Configuration

Main configuration file is located at `/etc/secban/mailer.yaml`.

### Configuration Options

| Option | Type | Description |
|---|---|---|
| `from` | string | Sender email address (`From:` header). |
| `to` | list of strings | List of recipient email addresses (`To:` header). |
| `socket_path` | string | UNIX domain socket path to listen on (e.g. `/var/run/secban/mailer.sock`). |
| `log_level` | string | Logging level (`trace`, `debug`, `info`, `warn`, `error`). Default: `info`. |
| `template_file` | string | Optional relative path to custom body template file inside configuration directory. Uses built-in template if empty. |
| `smtp.host` | string | SMTP server hostname or host:port (e.g. `mail.example.com` or `localhost:1025`). |
| `smtp.port` | integer | SMTP server port (default: `25`, overridden if port specified in `smtp.host`). |
| `smtp.username` | string | SMTP authentication username (can also be set via `SECBAN_MAILER_SMTP_USERNAME`). |
| `smtp.password` | string | SMTP authentication password (can also be set via `SECBAN_MAILER_SMTP_PASSWORD`). |
| `smtp.tls` | string | TLS policy: `notsl` (disable TLS), `mandatory` (require STARTTLS), or `opportunistic` (STARTTLS if supported; default). |

## Examples

### 1. Standard Configuration (`/etc/secban/mailer.yaml`)

```yaml
from: secban@example.com
log_level: info

smtp:
  host: mail.example.com:587
  username: "secban-alerts"
  password: "" # Set in environment variable SECBAN_MAILER_SMTP_PASSWORD
  tls: mandatory

socket_path: "/var/run/secban/mailer.sock"
template_file: ""

to:
  - security-team@example.com
  - ops@example.com
```

### 2. SecBan Agent Integration

To forward ban events from `secban-agent` to `secban-mailer`, add the mailer socket path to rule definitions in `/etc/secban/agent.yaml` or rule files under `/etc/secban/rules.d/`:

```yaml
rules_default:
  sockets:
    - /var/run/secban/mailer.sock
```

### 3. Custom Email Template

Specify `template_file: "custom_body.tpl"` in `/etc/secban/mailer.yaml`.

Template `/etc/secban/custom_body.tpl`:

```gotemplate
SecBan Alert: IP {{ $.IP }} has been banned.

Rule: {{ $.Rule }}
Timestamp: {{ $.Offender.Timestamp.Format "2006-01-02 15:04:05" }}
Incident Count: {{ $.Offender.Count }}

Recorded Entries:
{{- range $.Offender.Entries }}
  - [{{ .Timestamp.Format "2006-01-02 15:04:05" }}] {{ .Message }}
{{- end }}
```

### 4. Configuration Check (`secban-mailer-emailcheck`)

Validate configuration and send test alert email to configured recipients:

```sh
secban-mailer-emailcheck -config /etc/secban
```
