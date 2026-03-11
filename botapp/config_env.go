package botapp

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// loadDotEnv reads KEY=VALUE pairs from .env if present.
// Existing process environment variables have priority.
func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for lineNo := 1; s.Scan(); lineNo++ {
		k, v, ok, err := parseDotEnvLine(s.Text())
		if err != nil {
			return fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(k); exists {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return err
		}
	}
	return s.Err()
}

func parseDotEnvLine(raw string) (key, value string, ok bool, err error) {
	line := strings.TrimSpace(raw)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false, nil
	}
	if strings.HasPrefix(line, "export ") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
	}

	k, v, found := strings.Cut(line, "=")
	if !found {
		return "", "", false, nil
	}

	key = strings.TrimSpace(k)
	if key == "" {
		return "", "", false, fmt.Errorf("empty env key")
	}

	value, err = normalizeDotEnvValue(strings.TrimSpace(v))
	if err != nil {
		return "", "", false, err
	}

	return key, value, true, nil
}

func normalizeDotEnvValue(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	if strings.HasPrefix(v, "\"") {
		u, err := strconv.Unquote(v)
		if err != nil {
			return "", fmt.Errorf("invalid quoted value: %w", err)
		}
		return u, nil
	}
	if strings.HasPrefix(v, "'") {
		if len(v) < 2 || !strings.HasSuffix(v, "'") {
			return "", fmt.Errorf("invalid single-quoted value")
		}
		return v[1 : len(v)-1], nil
	}

	if i := strings.Index(v, " #"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v), nil
}
