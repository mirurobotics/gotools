package repoconfig

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const FileName = ".miru.yml"

type Config map[string]map[string]string

func Load(dir string) (Config, error) {
	path := filepath.Join(dir, FileName)
	//nolint:gosec // G304: trusted file path
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var raw map[string]map[string]*string
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	cfg := make(Config, len(raw))
	for section, values := range raw {
		if cfg[section], err = toSection(path, section, values); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

func toSection(
	path, section string, values map[string]*string,
) (map[string]string, error) {
	out := make(map[string]string, len(values))
	for name, value := range values {
		if value == nil {
			return nil, fmt.Errorf("%s: %s.%s: missing value", path, section, name)
		}
		out[name] = *value
	}
	return out, nil
}
