// Package discover suggests files a well-known tool owns that aren't
// tracked yet. It's a starting point, not a ceiling; nothing stops a user
// from tracking files the registry doesn't know about via "dotmgr add".
package discover

// Entry describes where a tool's config lives, per OS ($HOME-relative,
// "~/"-prefixed paths: files or directories).
type Entry struct {
	Name  string
	Paths map[string][]string // runtime.GOOS -> paths
}

var Registry = []Entry{
	{Name: "git", Paths: map[string][]string{
		"linux":  {"~/.gitconfig", "~/.config/git/config", "~/.gitignore_global"},
		"darwin": {"~/.gitconfig", "~/.config/git/config", "~/.gitignore_global"},
	}},
	{Name: "fish", Paths: map[string][]string{
		"linux":  {"~/.config/fish"},
		"darwin": {"~/.config/fish"},
	}},
	{Name: "zsh", Paths: map[string][]string{
		"linux":  {"~/.zshrc", "~/.zshenv", "~/.zprofile"},
		"darwin": {"~/.zshrc", "~/.zshenv", "~/.zprofile"},
	}},
	{Name: "bash", Paths: map[string][]string{
		"linux":  {"~/.bashrc", "~/.bash_profile", "~/.bash_aliases"},
		"darwin": {"~/.bashrc", "~/.bash_profile", "~/.bash_aliases"},
	}},
	{Name: "ssh-config", Paths: map[string][]string{
		"linux":  {"~/.ssh/config"},
		"darwin": {"~/.ssh/config"},
	}},
	{Name: "neovim", Paths: map[string][]string{
		"linux":  {"~/.config/nvim"},
		"darwin": {"~/.config/nvim"},
	}},
	{Name: "vim", Paths: map[string][]string{
		"linux":  {"~/.vimrc"},
		"darwin": {"~/.vimrc"},
	}},
	{Name: "tmux", Paths: map[string][]string{
		"linux":  {"~/.tmux.conf", "~/.config/tmux/tmux.conf"},
		"darwin": {"~/.tmux.conf", "~/.config/tmux/tmux.conf"},
	}},
	{Name: "starship", Paths: map[string][]string{
		"linux":  {"~/.config/starship.toml"},
		"darwin": {"~/.config/starship.toml"},
	}},
	{Name: "alacritty", Paths: map[string][]string{
		"linux":  {"~/.config/alacritty"},
		"darwin": {"~/.config/alacritty"},
	}},
	{Name: "kitty", Paths: map[string][]string{
		"linux":  {"~/.config/kitty"},
		"darwin": {"~/.config/kitty"},
	}},
	{Name: "gh", Paths: map[string][]string{
		"linux":  {"~/.config/gh/config.yml"},
		"darwin": {"~/.config/gh/config.yml"},
	}},
	{Name: "claude", Paths: map[string][]string{
		"linux":  {"~/.claude/CLAUDE.md", "~/.claude/settings.json", "~/.claude/settings.local.json", "~/.claude/keybindings.json"},
		"darwin": {"~/.claude/CLAUDE.md", "~/.claude/settings.json", "~/.claude/settings.local.json", "~/.claude/keybindings.json"},
	}},
	{Name: "ghostty", Paths: map[string][]string{
		"linux":  {"~/.config/ghostty"},
		"darwin": {"~/.config/ghostty", "~/Library/Application Support/com.mitchellh.ghostty"},
	}},
}
