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
| `dotmgr uninstall` | Remove dotmgr, keeping your current configs as regular files. See [Uninstalling](#uninstalling). |

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

## License

MIT, see [LICENSE](LICENSE).
