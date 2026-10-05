# Contributing

Contributions are welcome and encouraged.

To contribute fork this repository, make your changes, and open a pull request.

## Setup

Clone with submodules. The shared widget library ([dank-qml-common](https://github.com/AvengeMedia/dank-qml-common)) is vendored at `dank-qml-common/` and symlinked into `quickshell/DCommon`:

```bash
git clone --recurse-submodules https://github.com/AvengeMedia/dankfiles.git
# or, in an existing clone:
git submodule update --init
```

Install [prek](https://prek.j178.dev/) then activate pre-commit hooks:

```bash
prek install
```

The hooks run `gofmt -s`, `go vet`, `go mod tidy`, and the Go test suite for changes under `core/`, plus generic whitespace/YAML checks.

## Running

```bash
make run
# or with hot reload of the QML tree
DFILES_ENABLE_HOTRELOAD=1 core/bin/dfiles run -c quickshell
```

`dfiles ipc list` prints every IPC method the daemon exposes.

## Shared widgets (dank-qml-common)

Everything under `quickshell/DCommon/` is shared across the DMS suite and lives in the `dank-qml-common` submodule. It is a normal git worktree:

1. Edit files under `dank-qml-common/` (or through the `quickshell/DCommon` symlink, same files) and test in the running app. For isolated widget work, the library is its own runnable config with a gallery: `qs -p dank-qml-common`.
2. Commit and PR those changes in the `dank-qml-common` repo.
3. Once merged, bump the pointer here with `make update-common` and commit the submodule change together with `flake.lock`.

If a library change reads a new app-singleton property, add it to `quickshell/Common/Theme.qml` or `SettingsData.qml` in the same PR that bumps the pointer.

## Translations

See `quickshell/translations/README.md`.
