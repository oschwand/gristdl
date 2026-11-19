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

Precedence (highest first): flags, environment variables, `.env` file,
config file, defaults.

    gristdl connect --server https://grist.example.com --token XXX
    # environment / .env file (default path .env, loaded unless
    # variables are already set):
    GRIST_URL=https://grist.example.com
    GRIST_API_KEY=XXX
    GRIST_DOC_ID=tuSGBSshXrZgdeyxNfojAP
    GRIST_TEAM_ID=docs
    # or .gristdl.yaml / $HOME/.gristdl.yaml:
    server: https://grist.example.com
    token: XXX
    doc-id: tuSGBSshXrZgdeyxNfojAP
    team-id: docs

Use `--env-file PATH` (or `GRIST_ENV_FILE`) for a custom dotenv location,
`--env-file ""` to disable, and `--config PATH` for a custom config file.

## Commands

    gristdl login     # test auth against the server, list accessible orgs
    gristdl tree      # display all documents as a tree
