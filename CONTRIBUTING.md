# Contributing

Thank you for helping improve `go-zcashblob`. Contributions that make the wire
format implementation clearer, safer, or easier to verify are welcome.

## Prerequisites

- Go 1.22 or newer. Go 1.22 is the minimum version supported by the library.
- The current stable Go release for running the complete maintainer check.
- GNU Make is recommended, but is not required.

## Development workflow

1. Create a focused branch from the current `main` branch.
2. Make the smallest change that solves the problem.
3. Add or update tests for observable behavior.
4. Run the complete local preflight:

   ```sh
   make ci
   ```

5. Update documentation and `CHANGELOG.md` when users will notice the change.
6. Open a pull request that explains the motivation, compatibility impact, and
   verification performed.

Useful narrower commands are:

```sh
make test
make check
make fuzz-smoke
make bench
```

Longer fuzz runs can target either parser independently:

```sh
go test -run='^$' -fuzz='^FuzzParse$' -fuzztime=10m .
go test -run='^$' -fuzz='^FuzzCompactSize$' -fuzztime=10m .
```

Keep any useful fuzz failure corpus under `testdata/fuzz/` so the regression is
exercised by ordinary test runs.

### Without Make

On systems without Make, including a default Windows installation, run the Go
commands directly:

```text
gofmt -l .
go mod tidy -diff
go vet ./...
go test -race -shuffle=on -count=1 -covermode=atomic -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go test -run='^$' -fuzz='^FuzzParse$' -fuzztime=100000x -parallel=2 .
go test -run='^$' -fuzz='^FuzzCompactSize$' -fuzztime=100000x -parallel=2 .
```

`gofmt -l .` must produce no filenames. The statement total printed by
`go tool cover` must be at least 98%. The repository's `make check` command
generates the HTML and text coverage reports and enforces that threshold;
`make ci` adds both deterministic fuzz smoke targets.

## Tests and compatibility

- Prefer external-package examples for the public API and focused
  same-package tests for wire-format internals.
- Include fixed expected bytes or digests when a specification provides an
  independent oracle.
- Preserve parse/serialize round trips and complete writer-error propagation.
- Do not relax allocation limits or canonical-encoding checks without an
  explicit security rationale.
- Test the minimum supported Go version when changing compatibility-sensitive
  code.

Benchmarks are evidence, not acceptance gates. Include before-and-after results
when performance or allocations motivate a change.

## Pull requests

Keep each pull request reviewable and avoid unrelated cleanup. A pull request
should identify any change to:

- exported API or documented behavior;
- accepted or emitted wire encodings;
- defensive limits;
- transaction ID or authorization digest computation;
- minimum Go version.

All CI checks must pass. Maintainers may request a changelog entry, additional
test vectors, or a smaller change before merge.

## Security reports

Do not disclose an unpatched vulnerability in an issue or pull request. Follow
the private reporting instructions in [SECURITY.md](SECURITY.md).
