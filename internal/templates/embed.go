package templates

import (
	"embed"
	"path"
)

//go:embed files/config/.env.example files/config/php.ini files/lando/.lando.yml files/wp-cli.yml
var files embed.FS

func Read(name string) ([]byte, error) {
	return files.ReadFile(templatePath(name))
}

func templatePath(name string) string {
	cleaned := path.Clean(path.Clean("/" + name)[1:])
	if cleaned == "." {
		return "files"
	}

	return path.Join("files", cleaned)
}
