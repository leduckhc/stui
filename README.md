# stui

[![Website](https://img.shields.io/badge/website-stui.app-blue)](https://stui.app)
[![GitHub release](https://img.shields.io/github/v/release/natevick/stui)](https://github.com/natevick/stui/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A terminal user interface for browsing and downloading files from AWS S3, with full support for AWS IAM Identity Center (SSO).

![stui Demo](demo.gif)

## Features

- **Browse S3 buckets and prefixes** - Navigate your S3 storage like a file browser
- **AWS SSO support** - Works with IAM Identity Center profiles
- **Profile picker** - Select from available AWS profiles on startup
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
# Launch with profile picker
stui

# Launch with specific profile
stui --profile my-profile

# Launch directly into a bucket
stui --profile my-profile --bucket my-bucket

# Demo mode (no AWS credentials needed)
stui --demo
```

## Keyboard Shortcuts

### Navigation
| Key | Action |
|-----|--------|
| `↑/k`, `↓/j` | Move up/down |
| `Enter`, `→` | Open folder / Select |
| `Backspace`, `←` | Go back to parent folder |
| `PgUp/PgDn` | Page up/down |

In the **Browser**, `←`/`→` move through the folder hierarchy (and your cursor
position is remembered when you navigate back). In the other views they switch
tabs.

### Views
| Key | Action |
|-----|--------|
| `←/→` | Switch tabs (folder navigation in Browser) |
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
| `Esc` | Cancel / Close, or quit when there's nothing to cancel |
| `q` | Quit |

## Configuration

stui uses your standard AWS configuration (`~/.aws/config` and `~/.aws/credentials`).

### Example SSO Profile

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

## License

MIT License - see [LICENSE](LICENSE) for details.
