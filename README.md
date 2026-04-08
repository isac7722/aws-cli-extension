# awse — AWS CLI Extension

[![CI](https://github.com/isac7722/aws-cli-extension/actions/workflows/ci.yml/badge.svg)](https://github.com/isac7722/aws-cli-extension/actions/workflows/ci.yml)
[![Release](https://github.com/isac7722/aws-cli-extension/actions/workflows/release-please.yml/badge.svg)](https://github.com/isac7722/aws-cli-extension/actions/workflows/release-please.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/isac7722/aws-cli-extension)](https://goreportcard.com/report/github.com/isac7722/aws-cli-extension)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A lightweight CLI that extends AWS with **interactive profile management**, **SSM Parameter Store browsing**, **ECS deployment**, and **environment health checks** — all with a TUI-first experience.

## Features

- **Profile management** — Switch between AWS credential profiles with an interactive TUI selector; exports `AWS_PROFILE` to the current shell session
- **SSM Parameter Store** — Browse, create, update, and delete parameters with a tree-based TUI browser; `y` copies to clipboard; SecureString values masked by default
- **ECS deployment** — Interactively select cluster, service, and task definition to trigger force-new-deployment with optional stabilization wait
- **CLI doctor** — Check environment health (AWS CLI installed, credentials configured) with OS-appropriate installation guidance
- **Shell integration** — Hybrid shell function + binary design (like rbenv/pyenv) for seamless `AWS_PROFILE` switching

## Installation

### Homebrew (recommended)

```bash
brew install isac7722/awse/awse
```

### curl

```bash
curl -fsSL https://raw.githubusercontent.com/isac7722/aws-cli-extension/main/install.sh | bash
```

### Build from source

```bash
git clone https://github.com/isac7722/aws-cli-extension.git
cd aws-cli-extension
go build -o awse ./cmd/ae/
sudo mv awse /usr/local/bin/
```

### Shell integration

Add to your shell profile (`~/.zshrc` or `~/.bashrc`):

```bash
eval "$(awse init zsh)"   # for zsh
eval "$(awse init bash)"  # for bash
```

> The installer (`curl` and `brew`) sets this up automatically.

## Quick Start

```bash
awse doctor         # check your AWS environment
awse user           # switch AWS profile (interactive)
awse ssm            # browse SSM Parameter Store
awse ecs deploy     # deploy an ECS service
```

## Usage

### `awse user` — Profile Management

Manage AWS credential profiles stored in `~/.aws/credentials` and `~/.aws/config`.

```bash
awse user              # interactive profile switcher (default)
awse user list         # list all profiles
awse user add          # add a new profile interactively
awse user edit         # edit an existing profile
awse user delete       # delete a profile (with confirmation)
awse user switch       # alias for interactive switcher
```

After switching, a profile card is displayed:

```
✔ Switched to prod
╭──────────────────────────────╮
│ Profile    prod              │
│ Region     ap-northeast-2    │
│ Key ID     AKIA****WXYZ      │
│ Output     json              │
╰──────────────────────────────╯
```

### `awse ssm` — SSM Parameter Store

Browse and manage SSM Parameter Store parameters with an interactive TUI.

```bash
awse ssm                                           # interactive browser
awse ssm --profile prod --region ap-northeast-2    # skip selectors
awse ssm -p /app/prod                              # start from a path prefix
```

**Browser key bindings:**

| Key | Action |
|-----|--------|
| `j`/`k` or arrows | Navigate |
| `enter`/`space` | Expand folder / view parameter |
| `/` | Filter parameters |
| `y` | Copy value to clipboard |
| `n` | Create new parameter |
| `e` | Edit parameter value |
| `d` | Delete parameter |
| `v` | Reveal/hide SecureString value |
| `q`/`esc` | Quit |

Subcommands for scripting:

```bash
awse ssm get --name /app/db-host              # get a parameter value
awse ssm create --name /app/key --value val   # create a parameter
awse ssm put --name /app/key --value newval   # put (create/overwrite)
awse ssm update --name /app/key --value val2  # update existing
awse ssm delete --name /app/key               # delete a parameter
awse ssm batch-delete --prefix /app/old/      # delete by prefix
```

### `awse ecs deploy` — ECS Deployment

Interactively deploy ECS services with force-new-deployment.

```bash
awse ecs deploy                    # interactive (profile → region → cluster → service → task def)
awse ecs deploy --wait             # wait for service to stabilize
awse ecs deploy \
  --profile prod \
  --region ap-northeast-2 \
  --cluster my-cluster \
  --service my-service \
  --task-def my-task:42            # scripting mode (skip all selectors)
```

### `awse doctor` — Health Check

```bash
awse doctor
```

Checks AWS CLI v2 installation and displays OS-appropriate install guidance if missing.

### `awse uninstall` — Remove awse

```bash
awse uninstall            # interactive uninstall with confirmation
awse uninstall --dry-run  # preview what would be removed
```

Removes shell integration blocks from RC files. Does **not** touch `~/.aws/` credentials.

## Architecture

awse uses a hybrid **shell function + binary** design, similar to [rbenv](https://github.com/rbenv/rbenv) and [direnv](https://direnv.net/).

```
eval "$(awse init zsh)"

Shell wrapper function              Go binary
    │                                  │
    ├─ awse user switch ─────────────>│  TUI selector
    │    └─ capture AWSE_EXPORT:...    │  outputs AWSE_EXPORT:AWS_PROFILE=prod
    │    └─ export AWS_PROFILE=prod    │
    │                                  │
    ├─ awse ssm ─────────────────────>│  direct passthrough
    ├─ awse ecs deploy ──────────────>│
    └─ awse <anything> ─────────────>│
```

The wrapper intercepts `awse user switch` to capture the `AWSE_EXPORT:` protocol from stdout and export environment variables in the current shell. All other subcommands pass through directly.

## Contributing

```bash
git clone https://github.com/isac7722/aws-cli-extension.git
cd aws-cli-extension
bash scripts/setup-hooks.sh    # configure git hooks (goimports auto-format)
bash scripts/dev-install.sh    # build + install locally
```

Commits follow [Conventional Commits](https://www.conventionalcommits.org/) — releases are automated via [Release Please](https://github.com/googleapis/release-please).

## License

[MIT](LICENSE)
