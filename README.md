# GBT - Godot Build Tools

[![Build][build-badge]][build-workflow]
[![Go Version](https://img.shields.io/badge/go-1.21+-blue)](https://golang.org/)
[![Godot Version](https://img.shields.io/badge/godot-4.6%20%7C%20dev-blue)](https://godotengine.org/)
[![License](https://img.shields.io/badge/license-GPL--3.0--or--later-green)](LICENSE)


Cross-platform Godot build helper CLI.

I built this project as the Go rewrite of my personal `godot_build.py` workflow. The goal is to keep the same practical build, deploy, export-template, doctor, and CLI-shim behavior I use locally while producing a single binary that can run on Windows, macOS, and Linux.

> [!IMPORTANT]
> I am going to be straight with you, this is an opinionated build helper built around my personal Godot engine workflow. It is meant to be configurable, but it is not trying to be a generic build system abstraction to serve everyone. I am releasing it as
> as guesture to the community. You still need to install the compiler toolchains, SDKs, and third-party libraries required to build Godot on your platform. Start with the [Official Godot docs](https://docs.godotengine.org/en/latest/engine_details/development/compiling/compiling_for_windows.html) and make sure your environment matches the upstream requirements before expecting `gbt` to work.

> [!NOTE]
> The `doctor` command is there to show you what is missing or misconfigured in your local environment. It does not try to install toolchains, SDKs, Visual Studio workloads, or other Godot build dependencies for you.

> [!CAUTION]
> This workflow is Windows-only for now. The Go CLI is intended to support macOS and Linux later, but the real build flow, doctor checks, PATH handling, and shim behavior are still based on Windows assumptions today.

> [!TIP]
> Windows PATH handling is messy, especially once you mix user installs, per-shell environments, spaces in paths, and multiple engine binaries. That is why `gbt` creates user-level `.cmd` shims in a personal bin directory instead of expecting raw engine paths to behave nicely everywhere.

## GBT Commands

The Go CLI already covers the parts of the workflow I care about most:

- repo management: `pull`, `checkout`, `status`, `branches`
- build flows: `build`, `build-deploy`, `update`, `custom`, `clean`, `presets`
- deploy flows: `deploy`, `deploy-templates`, `list`, `info`
- environment checks: `doctor`
- binary and command installation: `install-self`, `install-cli`, `uninstall-cli`
- onboarding and config management: `onboard`, `config`

## What This Tool Is For

I use `gbt` to manage my local Godot engine builds without having to remember the exact repo, branch, build flags, deploy location, export-template install path, or CLI shim setup every time I install or build a version of Godot.

## Features

- standalone Go binary instead of a Python script
- per-user config stored in the OS-native config directory
- stable and dev deployment channels
- editor deploy plus export template install flow
- user-level CLI shims for `godot`, `godot-dev`, `godot-cs`, and `godot-dev-cs`
- first-run auto onboarding when no config file exists
- configurable repos, paths, default branches, and job count
- configurable personal bin directory with optional user PATH management
- self-install command for copying `gbt` into a user-writable bin directory

## Configuration

The CLI currently stores config in:

- Windows: `%AppData%\godot-build\config.toml`
- macOS: `~/Library/Application Support/godot-build/config.toml`
- Linux: `~/.config/godot-build/config.toml`

That path still uses `godot-build` for compatibility while the visible CLI command is `gbt`.

The generated starter config is intentionally generic. It will use your user-home-based paths plus the official Godot remote as default. Use the config commands to adjust to your needs and build folder layout

Example starter config:

```toml
```toml
[paths]
bin_dir = "C:\Users\<YourUsername>\bin"
build_root = "D:\Builds"
deploy_dir = "C:\Users\<YourUsername>\Engines\Godot\current"

[branches]
dev = "master"
stable = "4.6"

[defaults]
repo = "godot"
jobs = 8

[repos.godot]
git = "https://github.com/godotengine/godot.git"
path = "D:\Builds\godot"
```
```

On a real machine those `~` paths resolve to the current user's home directory. If your layout is different, change them with the `config` command.

## Onboarding

I wanted onboarding to be explicit, but also automatic on first run.

You can run it directly:

```powershell
gbt onboard
```

Or just run any command on a fresh machine. If no config file exists, `gbt` creates a starter config automatically and prints first-run guidance.

## Config Management

I wanted config management to be handled from the CLI instead of by editing TOML manually.

Show the resolved config:

```powershell
gbt config show
```

List editable keys:

```powershell
gbt config keys
```

Update a value:

```powershell
gbt config set paths.build_root D:\Builds
gbt config set paths.bin_dir C:\Users\you\bin
gbt config set defaults.repo godot
```

Clear a value:

```powershell
gbt config unset branches.stable
```

Manage repositories:

```powershell
gbt config repo add my-fork https://github.com/me/godot.git D:\Builds\godot-fork
gbt config repo remove my-fork
```

## Install GBT Itself

Use `install-self` to copy the currently running `gbt` binary into your configured personal bin directory.

Default install:

```powershell
gbt install-self
```

Use a one-off bin-dir override:

```powershell
gbt install-self --bin-dir C:\Users\you\bin
```

Skip PATH updates:

```powershell
gbt install-self --no-path
```

If an existing `gbt.exe` is already there, the tool warns and leaves it alone by default. To overwrite it intentionally:

```powershell
gbt install-self --force
```

If you want a custom bin dir to become the normal default, save it into config first:

```powershell
gbt config set paths.bin_dir C:\Users\you\bin
```

## Personal Bin And CLI Shims

The CLI shim flow is built around a user-writable personal bin directory so it does not require admin rights.

By default, `install-cli` uses the configured `paths.bin_dir` value and can add that directory to the user's PATH.

Install shims using the configured bin dir:

```powershell
gbt install-cli
gbt install-cli --mono
```

Use a one-off bin-dir override without changing saved config:

```powershell
gbt install-cli --bin-dir C:\Users\you\bin
gbt uninstall-cli --bin-dir C:\Users\you\bin
```

Skip PATH updates:

```powershell
gbt install-cli --no-path
```

If shim files already exist, the tool warns and leaves conflicting files alone by default. To overwrite those existing `.cmd` files intentionally:

```powershell
gbt install-cli --force
gbt install-cli --bin-dir C:\Users\you\bin --force
```

If you want a custom bin dir to become the normal default, save it into config:

```powershell
gbt config set paths.bin_dir C:\Users\you\bin
```

## Common Commands

Check local prerequisites:

```powershell
gbt doctor
```

Build and deploy the dev channel:

```powershell
gbt update
```

Build and deploy the stable channel:

```powershell
gbt update --stable
```

Build and deploy the stable C# editor:

```powershell
gbt update --stable --mono
```

Build a specific preset:

```powershell
gbt build editor
gbt build template-release-production
```

Install the binary and CLI shims after deploy:

```powershell
gbt install-self
gbt install-cli
gbt install-cli --mono
```

## Development

This is the normal local loop I use while working on the Go version:

```powershell
task check
task run
task onboard
task config
```

Or directly:

```powershell
go test ./...
go build -o gbt.exe ./cmd/godot-build
go run .\cmd\godot-build --help
```

## Notes

- The config file is the source of truth for user-specific paths and repo mappings.
- Only the official Godot remote is assumed by default.
- Any personal forks or alternate repo layouts should be added explicitly through `config repo add`.
- The CLI shim flow is designed around a user-level personal bin so it can avoid permission problems.
- `install-self` is meant to make `gbt` itself easy to place on PATH without admin rights.


[build-badge]: https://github.com/regiellis/godot-build-tools/actions/workflows/ci.yml/badge.svg
[build-workflow]: https://github.com/regiellis/godot-build-tools/actions/workflows/ci.yml
