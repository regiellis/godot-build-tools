# godot-build

Cross-platform Godot build helper CLI.

This repository is the Go rewrite of the local `godot_build.py` workflow. The goal is to keep the same practical build, deploy, template, doctor, and CLI-shim behavior while producing a single binary for Windows, macOS, and Linux.

## Goals

- Ship a standalone binary instead of a Python script.
- Support per-user config in the OS-native config directory.
- Keep paths configurable for:
  - user bin directory
  - build root
  - deploy directory
  - local source checkout
  - source git remote
- Provide sensible defaults for the official Godot repository.

## Planned Commands

- `doctor`
- `build`
- `deploy`
- `deploy-templates`
- `update`
- `install-cli`
- `uninstall-cli`
- `checkout`
- `pull`
- `config`

## Configuration

The CLI will load config from the user config directory:

- Windows: `%AppData%\godot-build\config.toml`
- macOS: `~/Library/Application Support/godot-build/config.toml`
- Linux: `~/.config/godot-build/config.toml`

Example:

```toml
[paths]
bin_dir = "C:\\Users\\playlogic\\bin"
build_root = "D:\\Builds"
deploy_dir = "D:\\Engines\\Godot\\current"

[repos.official]
git = "https://github.com/godotengine/godot.git"
path = "D:\\Builds\\godot"

[branches]
dev = "master"
stable = "4.6"

[defaults]
repo = "official"
jobs = 32
```

## Development

```powershell
go test ./...
go run .\cmd\godot-build
```

## Status

Initial scaffold only. The Python helper remains the behavioral reference.

