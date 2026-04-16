package templatesync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	sourceTemplatesDir = "templates"
	targetTemplatesDir = "internal/templates/files"
	wordPressDir       = "wordpress"
	dataVersionFile    = "DATA_VERSION"
	manifestFile       = "manifest.json"
)

var wordpressBackupCategories = map[string]struct{}{
	"database": {},
	"plugins":  {},
	"uploads":  {},
	"others":   {},
	"themes":   {},
}

type Manifest struct {
	Version string         `json:"version"`
	Files   []ManifestFile `json:"files"`
}

type ManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

func SyncRepo(repoRoot string) error {
	sourceRoot := filepath.Join(repoRoot, sourceTemplatesDir)
	targetRoot := filepath.Join(repoRoot, targetTemplatesDir)
	return Sync(sourceRoot, targetRoot)
}

func Sync(sourceRoot string, targetRoot string) error {
	if err := os.RemoveAll(targetRoot); err != nil {
		return err
	}
	if err := copyTreeExceptWordPress(sourceRoot, targetRoot); err != nil {
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

	targetWordPressRoot := filepath.Join(targetRoot, wordPressDir)
	if err := syncWordPressTemplates(wordPressRoot, targetWordPressRoot); err != nil {
		return err
	}

	manifest, err := BuildManifest(targetWordPressRoot)
	if err != nil {
		return err
	}

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

func EmbeddedDataVersion(repoRoot string) (string, error) {
	content, err := os.ReadFile(filepath.Join(repoRoot, targetTemplatesDir, wordPressDir, dataVersionFile))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

func copyTreeExceptWordPress(sourceRoot string, targetRoot string) error {
	return filepath.WalkDir(sourceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		if relative == wordPressDir || strings.HasPrefix(relative, wordPressDir+string(os.PathSeparator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
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

func syncWordPressTemplates(sourceRoot string, targetRoot string) error {
	return filepath.WalkDir(sourceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}

		targetRelative, err := normalizeWordPressRelativePath(relative)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetRoot, targetRelative)

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

func normalizeWordPressRelativePath(relative string) (string, error) {
	clean := filepath.Clean(relative)
	if clean == "." {
		return clean, nil
	}

	dir := filepath.Dir(clean)
	base := filepath.Base(clean)
	if dir == "." {
		category, ok, err := classifyLooseWordPressBackup(base)
		if err != nil {
			return "", err
		}
		if ok {
			return filepath.Join(category, base), nil
		}
		return clean, nil
	}

	firstSegment := strings.Split(filepath.ToSlash(clean), "/")[0]
	if _, ok := wordpressBackupCategories[firstSegment]; ok {
		return clean, nil
	}

	return clean, nil
}

func classifyLooseWordPressBackup(name string) (string, bool, error) {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch {
	case lower == "" || lower == ".gitkeep":
		return "", false, nil
	case strings.HasSuffix(lower, "-db.gz"), strings.HasSuffix(lower, "-db.sql"), lower == "db.sql", lower == "db.gz":
		return "database", true, nil
	case isNumberedArchive(lower, "plugins"):
		return "plugins", true, nil
	case isNumberedArchive(lower, "uploads"):
		return "uploads", true, nil
	case isNumberedArchive(lower, "others"):
		return "others", true, nil
	case isNumberedArchive(lower, "themes"):
		return "themes", true, nil
	case strings.HasSuffix(lower, ".zip"), strings.HasSuffix(lower, ".sql"), strings.HasSuffix(lower, ".gz"):
		return "", false, fmt.Errorf("unsupported wordpress backup file in templates/wordpress: %s", name)
	default:
		return "", false, nil
	}
}

func isNumberedArchive(name string, prefix string) bool {
	if !strings.HasSuffix(name, ".zip") {
		return false
	}
	stem := strings.TrimSuffix(name, ".zip")
	idx := strings.LastIndex(stem, "-"+prefix)
	if idx == -1 {
		return false
	}
	suffix := stem[idx+len(prefix)+1:]
	if suffix == "" {
		return true
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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
