# Contributing to chainview

Thanks for your interest! chainview is a watch-only, keyless EVM wallet monitor
for the terminal. This guide covers how to set up the environment, the code
style, and how to propose changes.

## Requirements

- **Go 1.25** or newer.
- `make` (optional but recommended; the targets just wrap the `go` commands).
- [`golangci-lint`](https://golangci-lint.run) **v2** to pass the linter locally
  (CI uses v2.12.2).

## Getting started

```sh
make setup    # go mod download + tidy
make run      # launch the TUI without building to disk
make build    # binary at ./bin/chainview with version info baked in
```

`chainview --help` lists the options; `chainview --version` prints the build
version. To diagnose, `chainview --debug` (or `CHAINVIEW_DEBUG=1`).

## Before opening a PR

Leave the tree green with the same checks CI runs:

```sh
gofmt -l .            # must print nothing
go vet ./...
make test             # go test -race ./...
make lint             # golangci-lint run
```

The TUI **golden tests** compare rendered frames. If you change the rendering on
purpose, regenerate the snapshots and review the diff before committing it:

```sh
go test ./internal/ui/ -run TestGolden -update
```

## Style

- Code formatted with `gofmt`; the linter (`.golangci.yml`) is the reference.
- The user-facing surface (UI strings, docs) is in **English**. Code comments
  are currently in Spanish; match the surrounding file. Comments explain the
  *why*, not the *what*.
- Errors lowercase and wrapped with context (`fmt.Errorf("...: %w", err)`).
- No API keys or dependencies that break the keyless-by-default startup.

## Commits and branches

- Work on a feature branch; PRs target `main`.
- Commit messages should be **natural and descriptive**, in the imperative
  ("Add…", "Fix…"). No type prefixes, no task IDs and no co-author lines. One
  commit per logical change.

## Design and decisions

Larger features are designed before being implemented. Specs live under `docs/`
(a local, untracked folder): they describe the goal, the decisions and the
implementation order. If you propose something substantial, open an issue first
to agree on the approach.

## Reporting bugs and requesting features

Use the issue templates. For bugs, include the version (`chainview --version`),
your OS/terminal and steps to reproduce. For vulnerabilities, please reach out
privately instead of opening a public issue.

## License

By contributing you agree that your contribution is published under the
project's [MIT](./LICENSE) license.
