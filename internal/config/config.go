package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Login  string
	Passwd string
}

// Load reads config from files then overrides with env vars.
// File search order: ./hfr.conf, ~/.config/hfr/config
func Load() *Config {
	cfg := &Config{}

	// Try config files in order
	paths := []string{"hfr.conf"}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "hfr", "config"))
	}

	for _, path := range paths {
		if readFile(path, cfg) {
			checkPerms(path)
			break
		}
	}

	// Env vars override
	if v := os.Getenv("HFR_LOGIN"); v != "" {
		cfg.Login = v
	}
	if v := os.Getenv("HFR_PASSWD"); v != "" {
		cfg.Passwd = v
	}

	return cfg
}

func readFile(path string, cfg *Config) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)

		switch k {
		case "login":
			cfg.Login = v
		case "passwd":
			cfg.Passwd = v
		}
	}
	return true
}

func checkPerms(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.Mode().Perm()&0077 != 0 {
		fmt.Fprintf(os.Stderr, "warning: %s is readable by others (chmod 600 recommended)\n", path)
	}
}
