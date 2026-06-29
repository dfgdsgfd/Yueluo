package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadEnvFiles imports .env values for local deployments while keeping real
// process environment variables authoritative.
func LoadEnvFiles() []string {
	loaded := []string{}
	for _, path := range candidateEnvPaths() {
		if loadEnvFile(path) {
			loaded = append(loaded, path)
		}
	}
	return loaded
}

func candidateEnvPaths() []string {
	candidates := []string{}
	add := func(path string) {
		if strings.TrimSpace(path) == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return
		}
		for _, existing := range candidates {
			if samePath(existing, abs) {
				return
			}
		}
		candidates = append(candidates, abs)
	}

	for _, key := range []string{"GIN_ENV_FILE", "ENV_FILE"} {
		for part := range strings.SplitSeq(os.Getenv(key), string(os.PathListSeparator)) {
			add(part)
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return candidates
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		add(filepath.Join(dir, "backend-gin", ".env"))
		add(filepath.Join(dir, ".env"))
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return candidates
}

func samePath(a string, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func loadEnvFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	values := map[string]string{}
	for scanner.Scan() {
		key, value, ok := parseEnvLine(scanner.Text())
		if !ok {
			continue
		}
		values[key] = value
	}
	for key, value := range values {
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
	return true
}

func parseEnvLine(line string) (string, string, bool) {
	text := strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
	if text == "" || strings.HasPrefix(text, "#") {
		return "", "", false
	}
	text = strings.TrimSpace(strings.TrimPrefix(text, "export "))
	index := strings.Index(text, "=")
	if index <= 0 {
		return "", "", false
	}
	key := strings.TrimSpace(text[:index])
	if !validEnvKey(key) {
		return "", "", false
	}
	value := strings.TrimSpace(text[index+1:])
	if len(value) >= 2 {
		quote := value[0]
		if (quote == '"' || quote == '\'') && value[len(value)-1] == quote {
			value = value[1 : len(value)-1]
			if quote == '"' {
				value = strings.NewReplacer(`\n`, "\n", `\r`, "\r", `\t`, "\t", `\"`, `"`, `\\`, `\`).Replace(value)
			}
		}
	}
	return key, value, true
}

func validEnvKey(key string) bool {
	for i, r := range key {
		if r == '_' || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9' && i > 0) {
			continue
		}
		return false
	}
	return key != ""
}
