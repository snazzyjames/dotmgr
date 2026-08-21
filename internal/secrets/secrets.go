// Package secrets guards against tracking private key material and
// credential files. It applies independently of the user's own
// .dotmgrignore, so a fresh repo is safe before that file has been edited.
package secrets

import "github.com/snazzyjames/dotmgr/internal/ignore"

var Denylist = ignore.Patterns{
	".ssh/id_*",
	".ssh/*.pem",
	".gnupg/private-keys-*",
	".aws/credentials",
	".claude/.credentials.json",
	"*.pem",
	"*_rsa",
	"*_ed25519",
}

// IsDenied reports whether rel (relative to $HOME) matches the built-in
// secrets denylist.
func IsDenied(rel string) bool {
	return Denylist.Match(rel)
}
