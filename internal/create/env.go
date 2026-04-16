package create

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const envFileName = ".env"
const globalConfigDirName = "toba"

func LoadEnvConfig() (ProjectConfig, error) {
	config, _, _, err := ResolveEnvConfig()
	return config, err
}

func ResolveEnvConfig() (ProjectConfig, string, bool, error) {
	envPath, err := GlobalEnvPath()
	if err != nil {
		return ProjectConfig{}, "", false, err
	}

	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return ProjectConfig{}, "", false, nil
	} else if err != nil {
		return ProjectConfig{}, "", false, err
	}

	values, err := loadEnvFile(envPath)
	if err != nil {
		return ProjectConfig{}, "", false, err
	}

	return ProjectConfig{
		Name:        values["TOBA_PROJECT_NAME"],
		PHPVersion:  values["TOBA_PHP_VERSION"],
		Database:    values["TOBA_DATABASE"],
		StarterRepo: values["TOBA_STARTER_REPO"],
		SSHTarget:   values["TOBA_SSH_TARGET"],
	}, envPath, false, nil
}

func GlobalEnvPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, globalConfigDirName, envFileName), nil
}

func CopyLocalEnvToGlobal(sourceDir string) (string, string, error) {
	sourcePath := filepath.Join(sourceDir, envFileName)
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", fmt.Errorf("no .env found in %s; run this command from the ToBA repository root", sourceDir)
		}
		return "", "", err
	}

	targetPath, err := GlobalEnvPath()
	if err != nil {
		return "", "", err
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return "", "", err
	}

	return sourcePath, targetPath, nil
}

func loadEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s:%d: invalid env entry", path, lineNo)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return values, nil
}
