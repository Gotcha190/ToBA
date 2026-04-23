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

// LoadEnvConfig loads the global ToBA configuration and returns only the
// resolved ProjectConfig value.
//
// Returns:
// - the resolved project configuration
// - an error when the global config path or file content cannot be read
func LoadEnvConfig() (ProjectConfig, error) {
	config, _, _, err := ResolveEnvConfig()
	return config, err
}

// ResolveEnvConfig loads the global ToBA configuration file and returns the
// parsed config together with source metadata.
//
// Returns:
// - the parsed project configuration
// - the global config path when the file exists
// - a reserved template flag, currently always false for resolved env files
// - an error when the path or file content cannot be read
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
		PHPVersion:          values["TOBA_PHP_VERSION"],
		Database:            values["TOBA_DATABASE"],
		StarterRepo:         values["TOBA_STARTER_REPO"],
		SSHTarget:           values["TOBA_SSH_TARGET"],
		RemoteWordPressRoot: values["TOBA_REMOTE_WORDPRESS_ROOT"],
	}, envPath, false, nil
}

// GlobalEnvPath returns the absolute path to the shared ToBA config file in
// the user's config directory.
//
// Returns:
// - the absolute config path
// - an error when the user config directory cannot be resolved
func GlobalEnvPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, globalConfigDirName, envFileName), nil
}

// ResolveGlobalEnvInitialization picks the content source for `toba config`
// and returns the source bytes together with source and target metadata.
//
// Parameters:
// - sourceDir: directory used to detect whether ToBA is running inside its own repository
//
// Returns:
// - the source file content that should be written
// - the logical or physical source path
// - the target global config path
// - whether the selected source is a template rather than a concrete repo config
// - an error when either source or target resolution fails
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

// WriteGlobalEnv writes the shared ToBA config file and converts permission
// failures into user-friendly messages.
//
// Parameters:
// - targetPath: absolute path of the global config file to create or overwrite
// - content: file content to write
//
// Returns:
// - an error when the parent directory cannot be created or the file cannot be written
//
// Side effects:
// - creates the target directory when needed
// - writes the global config file on disk
func WriteGlobalEnv(targetPath string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return wrapGlobalEnvPermissionError(targetPath, err)
	}
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return wrapGlobalEnvPermissionError(targetPath, err)
	}

	return nil
}

// resolveGlobalEnvSource chooses the best available source for initializing
// the shared config, preferring repository files over embedded defaults.
//
// Parameters:
// - sourceDir: directory used to check for repository-local .env files
//
// Returns:
// - the chosen source content
// - the chosen source path or logical embedded identifier
// - whether the chosen source is a template
// - an error when a candidate file cannot be read
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

// isTobaRepositoryRoot detects whether dir looks like the root of the ToBA
// repository by checking the module path and command entrypoint.
//
// Parameters:
// - dir: directory to inspect
//
// Returns:
// - true when dir appears to be the ToBA repository root
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

// wrapGlobalEnvPermissionError rewrites permission failures into clearer
// errors that include the target config path.
//
// Parameters:
// - targetPath: path that could not be written
// - err: original filesystem error
//
// Returns:
// - a friendlier permission error or the original error when it is not a permission issue
func wrapGlobalEnvPermissionError(targetPath string, err error) error {
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("cannot write global config at %s: permission denied", targetPath)
	}

	return err
}

// loadEnvFile parses a simple KEY=VALUE env file into a string map.
//
// Parameters:
// - path: env file path to read
//
// Returns:
// - a map of parsed key-value pairs
// - an error when the file cannot be opened or contains invalid entries
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
