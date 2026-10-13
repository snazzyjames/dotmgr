# dotmgr

A small CLI for tracking dotfiles in a git repo of your choosing and
symlinking them back into place on any machine.

`dotmgr` doesn't talk to GitHub, GitLab, or anywhere else directly: it shells
out to `git` against whatever remote URL you give it. GitHub is the natural
default (that's what these docs assume), but any git remote works.

## Install

Requires Go 1.27+ and `git` on `$PATH`.

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

dotmgr stores your dotfiles in a git repo that you own. That repo is the
source of truth that every machine syncs from.

1. Create a new, empty repo on GitHub (or any git host) to hold your dotfiles,
   e.g. `dotfiles`. Make it **private**: config files often contain hostnames,
   emails, and other details you may not want public. Don't add a README,
   license, or anything else; `dotmgr init` sets up the structure for you (see
   [How tracking works](#how-tracking-works)).
2. Pass the repo's clone URL to `init`, replacing `<your-username>` and
   `<your-repo>` with your own values:

```
dotmgr init git@github.com:<your-username>/<your-repo>.git
dotmgr add ~/.gitconfig
dotmgr add ~/.config/fish/config.fish
dotmgr push
```

Any URL `git clone` accepts works. The example above uses SSH, which needs an
SSH key added to your GitHub account but won't prompt for credentials on
`push`/`pull`. If you haven't set up SSH keys on this machine, use the HTTPS
URL instead:

```
dotmgr init https://github.com/<your-username>/<your-repo>.git
```

Pushing over HTTPS needs a personal access token or a credential helper
(e.g. `gh auth login`); GitHub doesn't accept your account password.

On a new machine, run `init` with the same repo URL, then `install`:

```
dotmgr init git@github.com:<your-username>/<your-repo>.git
dotmgr install
```

## Commands

| Command | Does |
|---|---|
| `dotmgr init <remote>` | Clone the dotfiles repo (it can be empty), write local config. |
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

`dotmgr init` clones your dotfiles repo to `~/.local/share/dotmgr/repo` (or
`$XDG_DATA_HOME/dotmgr/repo` if that's set). If the repo doesn't have them
yet, `init` creates a `files/` directory, a default `.dotmgrignore`, and a
short `README.md`. You don't need to create or arrange anything in the repo by
hand; `add` puts files in the right place, and `push` commits them.

After a few `add`s and a `push`, the repo looks like this. Everything under
`files/` mirrors its location relative to `$HOME`:

```
<your-repo>/
  files/
    .gitconfig                  ->  ~/.gitconfig
    .config/fish/config.fish    ->  ~/.config/fish/config.fish
  .dotmgrignore                 # patterns dotmgr won't add or suggest
  README.md
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

Every package under `internal/` is covered at 85%+. `cmd/dotmgr` is the one
exception: `main()` is a single `os.Exit(cli.Run(os.Args[1:]))` call, calling
it directly from a test would kill the test process, and it has no logic of
its own beyond what `internal/cli`'s tests already exercise, so it's left
untested rather than built out with a separate binary-coverage pipeline.

## License

MIT, see [LICENSE](LICENSE).
