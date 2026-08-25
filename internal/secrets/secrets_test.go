package secrets

import "testing"

func TestIsDenied(t *testing.T) {
	tests := []struct {
		rel  string
		want bool
	}{
		// Explicit denylist entries.
		{".ssh/id_rsa", true},
		{".ssh/id_ed25519", true},
		{".ssh/id_ed25519.pub", true}, // id_* also matches the public half
		{".ssh/server.pem", true},
		{".gnupg/private-keys-v1.d", true},
		{".aws/credentials", true},
		{".claude/.credentials.json", true},

		// Single-segment glob patterns matched against the basename,
		// regardless of directory.
		{"secret.pem", true},
		{"nested/dir/secret.pem", true},
		{"keys/deploy_rsa", true},
		{"keys/deploy_ed25519", true},

		// Should not match: ordinary config files.
		{".gitconfig", false},
		{".config/fish/config.fish", false},
		{".ssh/config", false},
		{".ssh/known_hosts", false},
		{".aws/config", false},

		// Similar-looking but distinct names shouldn't false-positive.
		{".ssh/identity", false}, // does not match "id_*" (no underscore)
		{"README.pem.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.rel, func(t *testing.T) {
			if got := IsDenied(tt.rel); got != tt.want {
				t.Errorf("IsDenied(%q) = %v, want %v", tt.rel, got, tt.want)
			}
		})
	}
}
