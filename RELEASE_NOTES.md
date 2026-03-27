# Release Notes

## GBT v0.1.0

This is the first tagged release of GBT - Godot Build Tools.

### Highlights

- Go-based rewrite of my personal Godot build workflow
- repo management commands for pull, checkout, status, and branches
- build, build-deploy, update, custom, clean, deploy, and deploy-templates flows
- stable and dev deployment channels
- export template install support
- user-level CLI shims for `godot`, `godot-dev`, `godot-cs`, and `godot-dev-cs`
- `install-self` for placing `gbt.exe` into a personal bin directory
- onboarding and config management from the CLI
- `doctor` checks for config, repo layout, and Windows toolchain expectations
- `version`, `which`, `config validate`, and global `--dry-run`
- app-level tests covering the new command and dry-run behavior

### Notes

- This workflow is Windows-only for now.
- The build assumptions are opinionated and centered around my personal Godot workflow.
- You still need the official Godot build dependencies installed locally.
