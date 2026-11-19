# gristdl

Download Grist documents via the Grist API.

## Build

    make build        # injects version from git describe
    ./gristdl version
    ./gristdl --version

Version comes from git tags via `-ldflags -X` (see `Makefile` and `.goreleaser.yml`).
Plain `go build` falls back to the VCS metadata embedded by the Go toolchain
(`dev-<shortsha>`), and `go install github.com/oschwand/gristdl@vX.Y.Z` reports
the module version. Releases: tag `vX.Y.Z` then `goreleaser release`.

## Configuration

Precedence (highest first): flags, environment variables (`GRISTDL_*`),
`.env` file, config file, defaults.

    gristdl connect --server https://grist.example.com --token XXX
    GRISTDL_TOKEN=xxx gristdl connect
    # .env (default path, loaded unless variables are already set):
    GRISTDL_SERVER=https://grist.example.com
    GRISTDL_TOKEN=XXX
    # or .gristdl.yaml / $HOME/.gristdl.yaml:
    server: https://grist.example.com
    token: XXX

Use `--env-file PATH` (or `GRISTDL_ENV_FILE`) for a custom dotenv location,
`--env-file ""` to disable, and `--config PATH` for a custom config file.

## Commands

    gristdl connect   # test auth against the server, list accessible orgs
