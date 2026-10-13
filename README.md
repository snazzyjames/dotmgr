# dotmgr

A small CLI for tracking dotfiles in a git repo of your choosing and
symlinking them back into place on any machine.

`dotmgr` doesn't talk to GitHub, GitLab, or anywhere else directly: it shells
out to `git` against whatever remote URL you give it. GitHub is the natural
default (that's what these docs assume), but any git remote works.

## Contents

- [Install](#install)
- [Build from source](#build-from-source)
- [Quickstart](#quickstart)
- [Commands](#commands)
  - [Command details](#command-details)
  - [Notes](#notes)
- [How tracking works](#how-tracking-works)
- [Updating](#updating)
- [Uninstalling](#uninstalling)
- [Testing](#testing)
- [Setting up an SSH key](#setting-up-an-ssh-key)
- [License](#license)

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
`push`/`pull` (see [Setting up an SSH key](#setting-up-an-ssh-key) if you've
never made one). If you'd rather skip SSH keys on this machine, use the HTTPS
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

| Command | Summary |
|---|---|
| [`dotmgr init <remote>`](#dotmgr-init) | Set up this machine with your dotfiles repo. |
| [`dotmgr add <path>`](#dotmgr-add) | Start tracking a file or directory. |
| [`dotmgr remove <path>`](#dotmgr-remove) | Stop tracking a file or directory. |
| [`dotmgr install`](#dotmgr-install) | Link every tracked file into place. |
| [`dotmgr relink <path>`](#dotmgr-relink) | Fix a link a program replaced, keeping this machine's version. |
| [`dotmgr status`](#dotmgr-status) | Show what's tracked and whether it's linked. |
| [`dotmgr scan`](#dotmgr-scan) | Suggest config files you might want to track. |
| [`dotmgr push`](#dotmgr-push) | Save your changes to the remote. |
| [`dotmgr pull`](#dotmgr-pull) | Get changes made on other machines. |
| [`dotmgr update`](#dotmgr-update) | Rebuild dotmgr and replace the installed copy. |
| [`dotmgr version`](#dotmgr-version) | Show which version is installed. |
| [`dotmgr uninstall`](#dotmgr-uninstall) | Remove dotmgr, keeping your configs. |

### Command details

#### dotmgr init

```
dotmgr init <remote>
```

Clones your dotfiles repo from `<remote>` to `~/.local/share/dotmgr/repo` and
saves the remote URL and clone location to `~/.config/dotmgr/config.toml`. If
the repo is empty or new, it also creates `files/`, a default `.dotmgrignore`
and a `README.md` in the clone. These aren't pushed until you run
`dotmgr push`. Run once per machine.

#### dotmgr add

```
dotmgr add [--force] <file-or-directory>
dotmgr add --link-dir <directory>
```

Moves the file at the given path (e.g. `~/.gitconfig`) into the clone at the
same location under `files/` (`files/.gitconfig`), then puts a symlink at the
original path pointing to it, so programs keep finding it where they expect.
The file is only on this machine until you run `dotmgr push`. Paths can be
absolute, start with `~/`, or be relative to the directory you're in, and
must be inside `$HOME`.

Given a directory (e.g. `dotmgr add ~/.config/kitty`), it does this for every
file inside it, including subdirectories. Each file gets its own symlink; the
directory itself stays a regular directory. Files matching `.dotmgrignore` or
the secrets denylist are skipped and listed. Files the program creates in
that directory later aren't tracked automatically: run the same `add` again
to pick them up. Files already tracked are skipped, so that's always safe.

`--force` adds a single file even if it matches the secrets denylist (see
[Notes](#notes)). It doesn't work on directories, so a whole directory can't
sweep secrets in by accident.

**`--link-dir`** tracks a directory as a whole instead: the directory itself
moves into the clone, and a single symlink replaces it. Use it for
directories that hold only config you write yourself, like `~/.config/nvim`
or `~/.config/kitty`. Compared to the default per-file mode:

- Files the program creates there later are tracked automatically, and
  deletes and renames sync too.
- Programs that save by replacing a file (see [`status`](#dotmgr-status))
  can't break the link.
- But everything the program writes there is synced, so it's a poor fit for
  directories that also hold caches, history, login tokens or other
  machine-specific state. Use per-file mode for those.

Safeguards: broad directories (`~/.config`, `~/.local/share`, `~/Library`,
`~/.ssh` and similar) can't be linked; a directory currently holding a file
on the secrets denylist is refused; and files that appear later and match
`.dotmgrignore` or the denylist are kept out of git by a generated block in
the repo's `.gitignore`. Files inside that were already tracked one by one
are converted automatically. On other machines, `install` links the
directory as a whole, backing up any directory the program already created
there.

#### dotmgr remove

```
dotmgr remove <file-or-directory>
```

The reverse of `add`: deletes the symlink and moves the file out of the
clone's `files/` back to that path as a regular file. Given a directory, it
does this for every tracked file inside it; given a directory linked with
`--link-dir`, it moves the whole directory back. Run `dotmgr push` afterward to
remove the files from the remote too; other machines keep their copies until
they're updated.

#### dotmgr install

```
dotmgr install [--dry-run]
```

Creates a symlink in `$HOME` for every file under the clone's `files/`,
creating any missing parent directories. Directories added with `--link-dir`
get a single symlink for the whole directory instead. If a different file is already at
that path, it's renamed to `<name>.bak.<timestamp>` first, never deleted.
Files already linked correctly are skipped, so it's safe to re-run. Use on a
new machine after `init`, and after `pull` to link files added on other
machines. `--dry-run` shows what would change.

#### dotmgr relink

```
dotmgr relink <path>
```

Fixes a tracked file that `status` reports as `unlinked` or `diverged`,
keeping this machine's version: the file at `<path>` replaces the repo's
copy, and the symlink goes back in its place. Run `dotmgr push` afterward.
If you'd rather keep the repo's version, use `dotmgr install` instead.

#### dotmgr status

```
dotmgr status
```

Read-only. Lists every tracked file (and every directory linked with
`--link-dir`) with its state on this machine, and explains how to fix any
problem it finds:

- `ok`: symlinked correctly.
- `missing`: nothing at that path in `$HOME` yet. Fix: `dotmgr install`.
- `unlinked`: a regular file with the same content as the repo's copy is
  where the symlink should be. Nothing is lost. Fix: `dotmgr relink <path>`.
- `diverged`: a regular file with *different* content is where the symlink
  should be, so its changes aren't reaching the repo. This usually means a
  program saved by writing a new file and renaming it over the link. To keep
  this machine's version, run `dotmgr relink <path>`. To keep the repo's
  version, run `dotmgr install`, which backs this one up first.
- `conflict`: something that isn't a file (e.g. a directory) is in the way.
  Fix: `dotmgr install`, which backs it up.

Then lists the same suggestions as `scan`.

#### dotmgr scan

```
dotmgr scan
```

Read-only. Looks in the usual config locations of tools dotmgr knows about
(git, fish, zsh, bash, ssh, neovim, vim, tmux, starship, alacritty, kitty, gh,
ghostty, Claude Code) and lists files that exist on this machine but aren't
tracked yet. It doesn't add anything; run `dotmgr add` on the ones you want.
Files matching `.dotmgrignore` or the secrets denylist are never suggested.

#### dotmgr push

```
dotmgr push
```

Commits every change in the clone (files added, removed or edited through
their symlinks) with the message `dotmgr: sync`, then pushes to your remote.
This is the only command that sends anything to the remote.

#### dotmgr pull

```
dotmgr pull
```

Runs `git pull` in the clone to fetch changes pushed from other machines.
Edits to files that are already linked take effect immediately. Files that
are new to this machine need `dotmgr install` to be linked into `$HOME`.

#### dotmgr update

```
dotmgr update [--version <version> | --from <checkout>]
```

Builds a new dotmgr in a temporary directory, then swaps it in for the
binary you ran (following symlinks to the real file). By default it builds the
latest published version with `go install`; `--version` picks a specific
one, and `--from` builds a local clone of this repo instead, which is handy
if you're working on dotmgr itself. `--from` refuses directories that aren't
a dotmgr checkout.

The swap happens only after a successful build, and it's atomic: the new
binary is written next to the old one and renamed over it. A failed build or
copy leaves the installed dotmgr unchanged. If the install directory isn't
writable, the new build is kept and `update` prints the `sudo install`
command to finish. Doesn't touch your dotfiles, the clone or the config. See
[Updating](#updating).

#### dotmgr version

```
dotmgr version
```

Prints the installed version. Builds from `go install ...@<version>` show
that version; builds from a checkout show the commit they were built from,
with `+dirty` if it had uncommitted changes.

#### dotmgr uninstall

```
dotmgr uninstall [--dry-run] [--restore-backups] [--yes] [--force]
```

Replaces every dotmgr symlink in `$HOME` with a regular copy of the file, then
deletes the clone and the config. Your configs stay as they are now (or go
back to their pre-dotmgr versions with `--restore-backups`), and the remote is
never touched. See [Uninstalling](#uninstalling).

### Notes

Paths above assume the defaults. If `$XDG_DATA_HOME` or `$XDG_CONFIG_HOME` is
set, the clone and the config live under those instead.

Because tracked files in `$HOME` are symlinks into the clone, editing
`~/.gitconfig` edits the clone's copy directly. There's no separate "save"
step; `dotmgr push` picks up the change.

`dotmgr add` checks a built-in secrets denylist (private keys, credential
files) before moving anything, independent of your own `.dotmgrignore`, and
`scan` never suggests those files. Use `dotmgr add --force` to override it for
one file.

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
    .config/kitty/              ->  ~/.config/kitty   (linked with --link-dir)
  .dotmgrignore                 # patterns dotmgr won't add or suggest
  .dotmgrdirs                   # directories linked as a whole
  .gitignore                    # generated block keeps junk out of linked dirs
  README.md
```

`dotmgr add ~/.gitconfig` moves it to `files/.gitconfig` and replaces the
original with a symlink back to it. `dotmgr install` walks `files/` and
recreates those symlinks under `$HOME` on whatever machine you run it on,
with one symlink per directory listed in `.dotmgrdirs`;
anything already in the way is backed up (renamed with a timestamp suffix),
never deleted.

## Updating

```
dotmgr update                        # latest published version
dotmgr update --version v1.2.0       # a specific version
dotmgr update --from ~/Code/dotmgr   # your local checkout
```

`update` rebuilds dotmgr with Go (so Go still needs to be on your `$PATH`)
and replaces the copy you ran, wherever it's installed. Check what you have
with `dotmgr version`.

If dotmgr is installed somewhere you need `sudo` to write to, like
`/usr/local/bin`, `update` builds the new version and then prints a
`sudo install ...` command to copy it into place. Running `sudo dotmgr update`
directly often fails, because `sudo` resets `$PATH` and can't find Go if it's
installed per-user (e.g. through mise, asdf or Homebrew).

## Uninstalling

`dotmgr uninstall` removes dotmgr from this machine. You choose what your
configs look like afterward:

| Command | Your configs afterward |
|---|---|
| `dotmgr uninstall` | **Exactly as they are right now.** Every symlink is replaced with a regular copy of the file it points to, including edits you haven't committed. Nothing about how your tools are configured changes. |
| `dotmgr uninstall --restore-backups` | **What this machine had before dotmgr**, for every file `dotmgr install` backed up. Files with no backup keep their current version. |

Add `--dry-run` to either one to see exactly what it would do without
changing anything:

```
dotmgr uninstall --dry-run
```

`uninstall` also shows that plan and asks before changing anything. Then it:

1. Replaces every dotmgr symlink in `$HOME` with a regular file: your current
   version, or the pre-dotmgr backup if you passed `--restore-backups`.
   Backups are the `<name>.bak.<timestamp>` files `dotmgr install` makes when
   something is in the way. If a file has several, the oldest (the original)
   is restored. Any backups not restored are left in place and listed so you
   can delete them.
2. Deletes the local clone and dotmgr's config.

If the local clone has changes you haven't pushed, `uninstall` stops and asks
you to run `dotmgr push` first so nothing is lost. Use `--force` to skip that
check. If any file can't be restored, it stops before deleting anything; fix
the error it prints and run it again.

Your remote dotfiles repo is never touched, so other machines keep working.
Two things are left for you to do yourself:

- **Delete the `dotmgr` binary.** `uninstall` prints the exact `rm` command.
- **Optional: delete the remote repo**, but only if no other machine still
  uses it. On GitHub: open the repo → Settings → scroll to the bottom → Delete
  this repository.

Directories that dotmgr created for your files (e.g. `~/.config/fish/`) are
left in place, since other programs may use them. Your SSH key isn't
something dotmgr made, so it isn't touched either.

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

## Setting up an SSH key

An SSH key lets git prove to GitHub that it's you, without a password. You
need one per machine. These steps are for macOS and Linux; GitHub's
[SSH docs](https://docs.github.com/en/authentication/connecting-to-github-with-ssh)
cover Windows and troubleshooting.

1. **Check whether you already have one.** If this lists a file ending in
   `.pub` (e.g. `id_ed25519.pub`), you can skip to step 4.

   ```
   ls ~/.ssh
   ```

2. **Generate a key.** The text after `-C` is just a label stored with the
   key, not a login. Something that names the machine works well, and it
   doesn't have to be your GitHub email. Press Enter to accept the default
   file location, then choose a passphrase (recommended) or leave it empty.

   ```
   ssh-keygen -t ed25519 -C "your-name@this-machine"
   ```

3. **Load the key so you aren't asked for the passphrase every time.**

   **macOS**

   This also saves the passphrase in your keychain:

   ```
   ssh-add --apple-use-keychain ~/.ssh/id_ed25519
   ```

   To keep the key loaded after a restart, add this to `~/.ssh/config`
   (create the file if it doesn't exist):

   ```
   Host github.com
     AddKeysToAgent yes
     UseKeychain yes
     IdentityFile ~/.ssh/id_ed25519
   ```

   **Linux**

   ```
   ssh-add ~/.ssh/id_ed25519
   ```

4. **Copy the public key** (the `.pub` file; never share the file without
   `.pub`):

   **macOS**

   This copies the key to your clipboard:

   ```
   pbcopy < ~/.ssh/id_ed25519.pub
   ```

   **Linux**

   This prints the key; select and copy the whole line it prints:

   ```
   cat ~/.ssh/id_ed25519.pub
   ```

5. **Add it to GitHub.** Go to GitHub → Settings → SSH and GPG keys → New SSH
   key, paste it in, and give it a title naming this machine.

6. **Test it.** The first time, you'll be asked whether to trust
   `github.com`; type `yes`. A message saying "You've successfully
   authenticated" means it worked.

   ```
   ssh -T git@github.com
   ```

You can now use the `git@github.com:...` URL with `dotmgr init`.

dotmgr never copies private keys (`~/.ssh/id_*`) into your dotfiles repo, so
each new machine needs its own key from these steps before it can clone over
SSH.

## License

MIT, see [LICENSE](LICENSE).
