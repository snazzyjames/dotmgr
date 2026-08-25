package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".dotmgrignore")
	content := "# a comment\n\n*.bak\n**/node_modules\n.aws/credentials\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	pats, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Patterns{"*.bak", "**/node_modules", ".aws/credentials"}
	if len(pats) != len(want) {
		t.Fatalf("Load() = %v, want %v", pats, want)
	}
	for i := range want {
		if pats[i] != want[i] {
			t.Errorf("pats[%d] = %q, want %q", i, pats[i], want[i])
		}
	}
}

func TestLoad_MissingFileIsEmptyNotError(t *testing.T) {
	pats, err := Load(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("Load of a missing file should not error, got: %v", err)
	}
	if len(pats) != 0 {
		t.Errorf("Load() = %v, want empty", pats)
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name string
		pats Patterns
		rel  string
		want bool
	}{
		{"literal path match", Patterns{".aws/credentials"}, ".aws/credentials", true},
		{"literal path no match", Patterns{".aws/credentials"}, ".aws/config", false},
		{"single segment glob matches basename anywhere", Patterns{"*.bak"}, "deep/nested/file.bak", true},
		{"single segment glob no match", Patterns{"*.bak"}, "deep/nested/file.txt", false},
		{"double-star prefix matches at any depth", Patterns{"**/node_modules"}, "a/b/node_modules", true},
		{"double-star prefix matches at root-ish depth", Patterns{"**/node_modules"}, "node_modules", true},
		{"double-star prefix no match", Patterns{"**/node_modules"}, "a/b/other", false},
		{"path glob with slash", Patterns{".config/*/secrets"}, ".config/app/secrets", true},
		{"path glob with slash no match", Patterns{".config/*/secrets"}, ".config/app/other", false},
		{"empty patterns never match", Patterns{}, "anything", false},
		{"backslash path normalized", Patterns{"*.bak"}, `deep\nested\file.bak`, true},
		{"backslash pattern normalized", Patterns{`deep/*/file.bak`}, "deep/nested/file.bak", true},
		{"multiple patterns, later one matches", Patterns{"*.pem", "*.bak"}, "f.bak", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pats.Match(tt.rel); got != tt.want {
				t.Errorf("Match(%q) with %v = %v, want %v", tt.rel, tt.pats, got, tt.want)
			}
		})
	}
}
