# stui

[![Website](https://img.shields.io/badge/website-stui.app-blue)](https://stui.app)
[![GitHub release](https://img.shields.io/github/v/release/natevick/stui)](https://github.com/natevick/stui/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A terminal user interface for browsing and downloading files from AWS S3, with full support for AWS IAM Identity Center (SSO).

![stui Demo](demo.gif)

## Features

- **Browse S3 buckets and prefixes** - Navigate your S3 storage like a file browser
- **AWS SSO support** - Works with IAM Identity Center profiles
- **S3-compatible storage** - MinIO, SeaweedFS, Ceph and more via mc aliases or stui endpoints
- **Profile picker** - Select from available AWS profiles, mc aliases and stui endpoints on startup
- **Multi-select** - Select multiple files/folders with spacebar
- **Download files** - Download individual files or entire prefixes
- **Sync folders** - Sync S3 prefixes to local directories (only downloads changed files)
- **Bookmarks** - Save frequently accessed locations
- **Demo mode** - Try the UI without AWS credentials

## Prerequisites

- **AWS CLI v2** - Required for SSO authentication
  - [Installation instructions](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
  - Verify installation: `aws --version`

## Installation

### Quick Install (macOS/Linux)

```bash
curl -fsSL https://stui.app/install.sh | bash
```

### From Binary

Download the latest release for your platform from the [Releases](https://github.com/natevick/stui/releases) page.

```bash
# macOS/Linux - make executable and move to PATH
chmod +x stui-*
sudo mv stui-* /usr/local/bin/stui

# Or without sudo, add to your local bin
mkdir -p ~/.local/bin
mv stui-* ~/.local/bin/stui
# Add to PATH in ~/.zshrc or ~/.bashrc: export PATH="$HOME/.local/bin:$PATH"
```

### From Source (requires Go)

```bash
go install github.com/natevick/stui/cmd/stui@latest
```

This installs to `$GOPATH/bin` (usually `~/go/bin`), which should be in your PATH.

Or clone and build:

```bash
git clone https://github.com/natevick/stui.git
cd stui
go build -o stui ./cmd/stui
```

## AWS SSO Login

Before using with SSO profiles, authenticate with the AWS CLI:

```bash
aws sso login --profile my-profile
```

This opens a browser window to complete authentication. Once logged in, you can use stui.

## Usage

```bash
# Launch with profile picker (lists aws profiles, mc aliases and stui endpoints)
stui

# Launch with specific profile
stui --profile my-profile

# Launch directly into a bucket
stui --profile my-profile --bucket my-bucket

# S3-compatible storage (MinIO / SeaweedFS / Ceph ...)
stui --alias my-minio                 # an mc alias from ~/.mc/config.json
stui --profile local --provider stui  # a stui endpoint, provider forced
stui --profile local --endpoint-url http://localhost:9000

# Demo mode (no AWS credentials needed)
stui --demo
```

### Flags
| Flag | Description |
|------|-------------|
| `--profile` | Profile (aws/stui) or alias (minio) name (also `AWS_PROFILE`) |
| `--alias` | Synonym of `--profile`, matching mc terminology |
| `--provider` | Force a provider: `aws`, `minio` or `stui` (skips name matching) |
| `--bucket` | Start directly in this bucket |
| `--endpoint-url` | Custom S3 endpoint URL (overrides the resolved one) |
| `--region` | AWS region (also `AWS_REGION`) |
| `--demo` | Run with mock data, no credentials needed |
| `--version` | Print version and exit |

## Keyboard Shortcuts

### Navigation
| Key | Action |
|-----|--------|
| `↑/k`, `↓/j` | Move up/down |
| `Enter` | Open folder / Select |
| `Backspace` | Go back |
| `PgUp/PgDn` | Page up/down |

### Views
| Key | Action |
|-----|--------|
| `←/→` | Switch tabs |
| `Tab` | Next tab |
| `Shift+Tab` | Previous tab |
| `1/2/3` | Jump to tab |

### Actions
| Key | Action |
|-----|--------|
| `Space` | Select/deselect item |
| `d` | Download selected |
| `s` | Sync prefix to local |
| `b` | Add bookmark |
| `r` | Refresh |
| `/` | Filter list |

### General
| Key | Action |
|-----|--------|
| `?` | Toggle help |
| `Esc` | Cancel / Close |
| `q` | Quit |

## Configuration

stui reads connection targets from three independent sources ("providers"). It
never modifies or extends the AWS config schema.

| Provider | Config source | Identifier |
|----------|---------------|------------|
| `aws` | `~/.aws/config` (+ `~/.aws/credentials`) | profile |
| `minio` | `~/.mc/config.json` (or `$MC_CONFIG_DIR`) | alias |
| `stui` | `~/.config/stui/config.json` | profile |

A name given with `--profile`/`--alias` is matched within a **single** provider
— there is never any fallback between providers. If the same name exists in more
than one provider, stui asks you to disambiguate with `--provider`.

### Example SSO Profile (`aws`)

```ini
[sso-session my-sso]
sso_start_url = https://my-company.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access

[profile my-profile]
sso_session = my-sso
sso_account_id = 123456789012
sso_role_name = MyRole
region = us-west-2
```

### S3-compatible storage (MinIO, SeaweedFS, Ceph, ...)

If you already use the MinIO client, your `mc` aliases work out of the box:

```bash
mc alias set my-minio http://localhost:9000 ACCESS_KEY SECRET_KEY
stui --alias my-minio
```

Or define endpoints in stui's own config at `~/.config/stui/config.json`
(written with `0600` permissions — keep it that way, it can hold secrets):

```json
{
  "endpoints": {
    "seaweed": {
      "endpoint_url": "http://localhost:8333"
    },
    "minio": {
      "endpoint_url": "http://localhost:9000",
      "access_key_id": "ACCESS_KEY",
      "secret_access_key": "SECRET_KEY",
      "path_style": true,
      "region": "us-east-1"
    }
  }
}
```

When credentials are omitted, stui falls back to the AWS default credential
chain (env vars, etc.). Path-style addressing is enabled automatically for
custom endpoints unless `path_style` is set to `false`.

## License

MIT License - see [LICENSE](LICENSE) for details.
