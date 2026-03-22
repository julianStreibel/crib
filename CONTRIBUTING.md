# Contributing to crib

Thanks for wanting to contribute! Here's how to get started.

## Setup

```bash
git clone https://github.com/julianStreibel/crib.git
cd crib
go build -o crib .
```

You'll need Go 1.24+ installed.

## Making changes

1. Fork the repo
2. Create a branch from `main`: `git checkout -b my-feature`
3. Make your changes
4. Run tests: `go test ./...`
5. Build: `go build -o crib .`
6. Open a PR against `main`

## Code style

- Run `golangci-lint run` before submitting (CI checks this)
- Keep it simple. Less code is better
- Error messages should be helpful to LLM agents (include hints and alternatives)
- Use the structured error types from `internal/errors/`

## Adding a new integration

1. Create `internal/<name>/` with your client code
2. Add CLI commands in `cmd/<name>.go`
3. Add config loading in `internal/config/config.go`
4. Add a setup step in `cmd/setup.go` (make it optional and skippable)
5. Update the plugin skill in `plugin/skills/home/SKILL.md`
6. Update `README.md`

Look at `internal/tradfri/` or `internal/sonos/` as reference implementations.

### Design principles

- **Generic commands for generic actions**: `crib devices on`, `crib speakers play` work across all providers
- **Provider-specific commands for provider-specific features**: `crib sonos group` stays under the provider name
- **Fuzzy name matching**: users and agents should be able to use readable names, not IDs
- **Structured errors**: every error should tell the caller what went wrong and what to do next
- **No config needed for discovery-based integrations**: Sonos auto-discovers, no setup required

## Testing

If you have the actual hardware, test against real devices. If not, make sure existing tests pass and your code builds cleanly.

```bash
go test ./...
go build ./...
```

## Questions?

Open an issue or start a discussion. Happy to help.
