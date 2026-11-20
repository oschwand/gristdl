# gristdl

Download Grist documents via the Grist API.

## Build

    make build        # or plain go build
    ./gristdl version
    ./gristdl --version

Version information comes from the build metadata embedded by the Go
toolchain (`debug.ReadBuildInfo`), with no external injection:
`go install github.com/oschwand/gristdl@vX.Y.Z` reports the module
version, a local `go build` reports `devel+<commit>[-dirty]`, and a
binary built outside version control reports `(unknown)`. Releases: tag
`vX.Y.Z` then `goreleaser release`.

## Configuration

Precedence (highest first): flags, environment variables, config file,
defaults. The config file is a dotenv file loaded from the XDG config
directory, usually `~/.config/gristdl/config.env`:

    GRIST_URL=https://grist.example.com
    GRIST_API_KEY=XXX
    GRIST_DOC_ID=tuSGBSshXrZgdeyxNfojAP
    GRIST_TEAM_ID=docs

Real environment variables always take precedence over the config file.
Use `--env-file PATH` (or `GRIST_ENV_FILE`) for a custom location and
`--env-file ""` to disable loading entirely.

## Commands

    gristdl login     # test auth against the server, list accessible orgs
    gristdl tree      # display all documents as a tree
    gristdl pull [-o DIR] [--force]   # download all documents into DIR

## Pull

`pull` downloads every accessible document as a `.grist` file into a
directory hierarchy mirroring the server:

    DIR/
      <Org>/
        <Workspace>/
          <Document>.grist

The personal org ("Personal") does not get its own folder: its
workspaces sit directly under DIR.

Progress is tracked in `DIR/.gristdl-pull.json`, which maps each document
ID (its immutable UUID) to its local path and last-seen modification time.
This makes subsequent pulls cheap:

- unchanged documents are skipped,
- a document renamed or moved on the server is renamed or moved locally
  instead of downloaded again,
- a document modified on the server (its `updatedAt` changed) is
  re-downloaded,
- documents deleted on the server are kept on disk.

Existing files are never overwritten: if a target file exists and was not
placed there by a previous pull, the document is refused with an error.
`--force` overwrites existing files and re-downloads up-to-date documents.

## AI Disclosure

Most of the code, tests, and documentation in this repository were
written with AI assistance (Crush, running the GLM model by Z.ai),
under the direction and review of the author. Commits carry the
corresponding machine-generated attribution lines. The author retains
copyright and full responsibility for the content.

## Licence

MIT — see [LICENSE](LICENSE).
