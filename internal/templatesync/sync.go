package templatesync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	sourceTemplatesDir = "templates"
	targetTemplatesDir = "internal/templates/files"
	wordPressDir       = "wordpress"
	dataVersionFile    = "DATA_VERSION"
	manifestFile       = "manifest.json"
	configDirName      = "toba"
)

var backupTypePattern = regexp.MustCompile(`-(db|plugins\d*|uploads\d*|others\d*)\.(zip|gz)$`)

type Manifest struct {
	Version string         `json:"version"`
	Files   []ManifestFile `json:"files"`
}

type ManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type BackupFileInfo struct {
	SourcePath string
	FileName   string
	Slot       string
	Category   string
	TargetDir  string
	TargetPath string
}

func SyncRepo(repoRoot string) error {
	sourceRoot := filepath.Join(repoRoot, sourceTemplatesDir)
	targetRoot := filepath.Join(repoRoot, targetTemplatesDir)
	return Sync(sourceRoot, targetRoot)
}

func InstallOverrideBackup(sourcePath string) (BackupFileInfo, string, error) {
	info, err := DetectBackupFile(sourcePath)
	if err != nil {
		return BackupFileInfo{}, "", err
	}

	overrideRoot, err := OverrideTemplatesDir()
	if err != nil {
		return BackupFileInfo{}, "", err
	}

	targetDir := filepath.Join(overrideRoot, info.TargetDir)
	targetPath := filepath.Join(targetDir, info.FileName)

	for _, existing := range matchingTemplateFiles(targetDir, info.Slot) {
		if err := os.Remove(existing); err != nil {
			return BackupFileInfo{}, "", err
		}
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return BackupFileInfo{}, "", err
	}

	input, err := os.Open(info.SourcePath)
	if err != nil {
		return BackupFileInfo{}, "", err
	}
	defer input.Close()

	output, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return BackupFileInfo{}, "", err
	}
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		return BackupFileInfo{}, "", err
	}

	version, err := writeOverrideManifest(overrideRoot)
	if err != nil {
		return BackupFileInfo{}, "", err
	}

	info.TargetPath = targetPath
	return info, version, nil
}

func Sync(sourceRoot string, targetRoot string) error {
	if err := os.RemoveAll(targetRoot); err != nil {
		return err
	}
	if err := copyTree(sourceRoot, targetRoot); err != nil {
		return err
	}

	wordPressRoot := filepath.Join(sourceRoot, wordPressDir)
	exists, err := dirExists(wordPressRoot)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	manifest, err := BuildManifest(wordPressRoot)
	if err != nil {
		return err
	}

	targetWordPressRoot := filepath.Join(targetRoot, wordPressDir)
	if err := os.MkdirAll(targetWordPressRoot, 0755); err != nil {
		return err
	}

	versionPath := filepath.Join(targetWordPressRoot, dataVersionFile)
	if err := os.WriteFile(versionPath, []byte(manifest.Version+"\n"), 0644); err != nil {
		return err
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(targetWordPressRoot, manifestFile)
	if err := os.WriteFile(manifestPath, append(manifestBytes, '\n'), 0644); err != nil {
		return err
	}

	return nil
}

func BuildManifest(wordPressRoot string) (Manifest, error) {
	var manifest Manifest

	err := filepath.WalkDir(wordPressRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relative, err := filepath.Rel(wordPressRoot, path)
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		hash, err := fileSHA256(path)
		if err != nil {
			return err
		}

		manifest.Files = append(manifest.Files, ManifestFile{
			Path:   filepath.ToSlash(relative),
			SHA256: hash,
			Size:   info.Size(),
		})

		return nil
	})
	if err != nil {
		return Manifest{}, err
	}

	sort.Slice(manifest.Files, func(i, j int) bool {
		return manifest.Files[i].Path < manifest.Files[j].Path
	})

	versionHash := sha256.New()
	for _, file := range manifest.Files {
		if _, err := io.WriteString(versionHash, file.Path); err != nil {
			return Manifest{}, err
		}
		if _, err := io.WriteString(versionHash, ":"); err != nil {
			return Manifest{}, err
		}
		if _, err := io.WriteString(versionHash, file.SHA256); err != nil {
			return Manifest{}, err
		}
		if _, err := io.WriteString(versionHash, "\n"); err != nil {
			return Manifest{}, err
		}
	}

	fullVersion := hex.EncodeToString(versionHash.Sum(nil))
	if len(fullVersion) < 12 {
		return Manifest{}, fmt.Errorf("invalid data version hash")
	}
	manifest.Version = fullVersion[:12]

	return manifest, nil
}

func DetectBackupFile(sourcePath string) (BackupFileInfo, error) {
	absolutePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return BackupFileInfo{}, err
	}

	fileName := filepath.Base(absolutePath)
	match := backupTypePattern.FindStringSubmatch(strings.ToLower(fileName))
	if match == nil {
		return BackupFileInfo{}, fmt.Errorf("unsupported Updraft backup name: %s", fileName)
	}

	slot := match[1]
	category := categoryForSlot(slot)
	if category == "" {
		return BackupFileInfo{}, fmt.Errorf("unsupported Updraft backup type: %s", slot)
	}

	targetDir := filepath.Join(wordPressDir, category)

	return BackupFileInfo{
		SourcePath: absolutePath,
		FileName:   fileName,
		Slot:       slot,
		Category:   category,
		TargetDir:  targetDir,
	}, nil
}

func EmbeddedDataVersion(repoRoot string) (string, error) {
	content, err := os.ReadFile(filepath.Join(repoRoot, targetTemplatesDir, wordPressDir, dataVersionFile))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

func OverrideTemplatesDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, configDirName, "templates"), nil
}

func copyTree(sourceRoot string, targetRoot string) error {
	return filepath.WalkDir(sourceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetRoot, relative)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		return copyFile(path, targetPath, info.Mode().Perm())
	})
}

func copyFile(sourcePath string, targetPath string, mode os.FileMode) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return err
	}

	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func dirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func writeOverrideManifest(overrideRoot string) (string, error) {
	wordPressRoot := filepath.Join(overrideRoot, wordPressDir)
	exists, err := dirExists(wordPressRoot)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("override directory does not exist: %s", wordPressRoot)
	}

	manifest, err := BuildManifest(wordPressRoot)
	if err != nil {
		return "", err
	}

	versionPath := filepath.Join(wordPressRoot, dataVersionFile)
	if err := os.WriteFile(versionPath, []byte(manifest.Version+"\n"), 0644); err != nil {
		return "", err
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}

	manifestPath := filepath.Join(wordPressRoot, manifestFile)
	if err := os.WriteFile(manifestPath, append(manifestBytes, '\n'), 0644); err != nil {
		return "", err
	}

	return manifest.Version, nil
}

func categoryForSlot(slot string) string {
	switch {
	case slot == "db":
		return "database"
	case strings.HasPrefix(slot, "plugins"):
		return "plugins"
	case strings.HasPrefix(slot, "uploads"):
		return "uploads"
	case strings.HasPrefix(slot, "others"):
		return "others"
	default:
		return ""
	}
}

func matchingTemplateFiles(targetDir string, slot string) []string {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return nil
	}

	var matches []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := DetectBackupFile(filepath.Join(targetDir, entry.Name()))
		if err != nil {
			continue
		}
		if info.Slot == slot {
			matches = append(matches, filepath.Join(targetDir, entry.Name()))
		}
	}

	return matches
}
