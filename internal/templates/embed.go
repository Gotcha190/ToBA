package templates

import (
	"embed"
	"path"
)

//go:embed files/config/.env.example files/config/php.ini files/lando/.lando.yml files/wp-cli.yml
var files embed.FS

// Read returns an embedded template file by its logical name.
//
// Parameters:
// - name: logical template path such as `config/php.ini` or `wp-cli.yml`
//
// Returns:
// - the embedded template bytes
// - an error when the requested template does not exist
func Read(name string) ([]byte, error) {
	return files.ReadFile(templatePath(name))
}

// templatePath normalizes a logical template name into the embedded files/
// namespace.
//
// Parameters:
// - name: logical template path requested by the caller
//
// Returns:
// - the canonical embedded path under `files/`
func templatePath(name string) string {
	cleaned := path.Clean(path.Clean("/" + name)[1:])
	if cleaned == "." {
		return "files"
	}

	return path.Join("files", cleaned)
}
