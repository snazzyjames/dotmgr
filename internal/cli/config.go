package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/snazzyjames/dotmgr/internal/config"
)

func loadConfig() (*config.Config, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("not initialized yet — run: dotmgr init <remote>")
		}
		return nil, err
	}
	return cfg, nil
}
