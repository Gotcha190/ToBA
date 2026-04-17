package create

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotcha190/toba/internal/templates"
)

const envFileName = ".env"
const envExampleFileName = ".env.example"
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

func ResolveGlobalEnvInitialization(sourceDir string) ([]byte, string, string, bool, error) {
	targetPath, err := GlobalEnvPath()
	if err != nil {
		return nil, "", "", false, err
	}

	content, sourcePath, fromTemplate, err := resolveGlobalEnvSource(sourceDir)
	if err != nil {
		return nil, "", "", false, err
	}

	return content, sourcePath, targetPath, fromTemplate, nil
}

func WriteGlobalEnv(targetPath string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return wrapGlobalEnvPermissionError(targetPath, err)
	}
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return wrapGlobalEnvPermissionError(targetPath, err)
	}

	return nil
}

func resolveGlobalEnvSource(sourceDir string) ([]byte, string, bool, error) {
	if isTobaRepositoryRoot(sourceDir) {
		for _, fileName := range []string{envFileName, envExampleFileName} {
			sourcePath := filepath.Join(sourceDir, fileName)
			content, err := os.ReadFile(sourcePath)
			if err == nil {
				return content, sourcePath, fileName == envExampleFileName, nil
			}
			if !os.IsNotExist(err) {
				return nil, "", false, err
			}
		}
	}

	content, err := templates.Read("config/.env.example")
	if err != nil {
		return nil, "", false, err
	}

	return content, "embedded:.env.example", true, nil
}

func isTobaRepositoryRoot(dir string) bool {
	goModPath := filepath.Join(dir, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		return false
	}
	if !strings.Contains(string(goModContent), "github.com/gotcha190/toba") {
		return false
	}

	if _, err := os.Stat(filepath.Join(dir, "cmd", "root.go")); err != nil {
		return false
	}

	return true
}

func wrapGlobalEnvPermissionError(targetPath string, err error) error {
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("cannot write global config at %s: permission denied", targetPath)
	}

	return err
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
