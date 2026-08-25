# dotmgr

A small CLI for tracking dotfiles in a git repo of your choosing and
symlinking them back into place on any machine.

`dotmgr` doesn't talk to GitHub, GitLab, or anywhere else directly: it shells
out to `git` against whatever remote URL you give it. GitHub is the natural
default (that's what these docs assume), but any git remote works.

## Install

Requires Go 1.22+ and `git` on `$PATH`.

```
go install github.com/snazzyjames/dotmgr/cmd/dotmgr@latest
```

## Build from source

```
git clone https://github.com/snazzyjames/dotmgr.git
cd dotmgr
go build -o dotmgr ./cmd/dotmgr
```

This produces a `dotmgr` binary in the current directory. Move it onto your
`$PATH` (e.g. `sudo mv dotmgr /usr/local/bin/`), or run `go install ./cmd/dotmgr`
from the repo root to install it to `$GOBIN` instead.

## Quickstart

```
dotmgr init git@github.com:you/dotfiles.git
dotmgr add ~/.gitconfig
dotmgr add ~/.config/fish/config.fish
dotmgr push
```

On a new machine:

```
dotmgr init git@github.com:you/dotfiles.git
dotmgr install
```

## Commands

| Command | Does |
|---|---|
| `dotmgr init <remote>` | Clone (or create) the dotfiles repo, write local config. |
| `dotmgr add <path>` | Move a file into the repo, symlink it back. |
| `dotmgr remove <path>` | Reverse of add. |
| `dotmgr install` | Symlink every tracked file into place; back up conflicts. |
| `dotmgr status` | Show tracked files, broken links, and known-tool configs not yet tracked. |
| `dotmgr scan` | Suggest untracked files a known tool owns. |
| `dotmgr push` / `dotmgr pull` | Commit-and-push / pull the local clone. |

Every command that adds a file (`add` and `scan`) checks a built-in secrets
denylist first (private keys, credential files) independent of your own
`.dotmgrignore`. Use `dotmgr add --force` to override it for one file.

## How tracking works

Your dotfiles repo mirrors `$HOME` under `files/`:

```
<your-dotfiles-repo>/
  files/
    .gitconfig
    .config/fish/config.fish
  .dotmgrignore
```

`dotmgr add ~/.gitconfig` moves it to `files/.gitconfig` and replaces the
original with a symlink back to it. `dotmgr install` walks `files/` and
recreates those symlinks under `$HOME` on whatever machine you run it on;
anything already in the way is backed up (renamed with a timestamp suffix),
never deleted.

## Testing

```
go test ./...
```

To see a per-package coverage summary in the terminal:

```
go test ./... -cover
```

For a browsable, line-by-line HTML report:

```
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

Then open `coverage.html` in a browser. `coverage.out` and `coverage.html`
are build artifacts, not checked in.

Coverage today is concentrated in `internal/store`, `internal/secrets`, and
`internal/linker`; `internal/config`, `internal/ignore`, `internal/discover`,
and `internal/cli` don't have tests yet. This section will grow as that
changes.

## License

MIT, see [LICENSE](LICENSE).
