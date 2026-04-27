package wordpress

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
)

// BackupSourceURL reads the original site URL from SQL backup metadata.
//
// Parameters:
// - sqlPath: SQL dump path to inspect
//
// Returns:
// - the validated source URL found in the backup comments
// - an error when the file cannot be read or the URL metadata is missing/invalid
func BackupSourceURL(sqlPath string) (string, error) {
	file, err := os.Open(sqlPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	var backupURL string
	parseLine := func(line string) error {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "# Home URL:"):
			value, err := validateURL(strings.TrimSpace(strings.TrimPrefix(line, "# Home URL:")))
			if err != nil {
				return err
			}
			backupURL = value
			return errStopReading
		case backupURL == "" && strings.HasPrefix(line, "# Backup of:"):
			backupURL = strings.TrimSpace(strings.TrimPrefix(line, "# Backup of:"))
		}
		return nil
	}

	err = scanMetadataHeader(file, parseLine)
	if err != nil && !errors.Is(err, errStopReading) {
		return "", err
	}
	if backupURL != "" {
		return validateURL(backupURL)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	err = forEachLine(file, parseLine)
	if err != nil && !errors.Is(err, errStopReading) {
		return "", err
	}
	if backupURL == "" {
		return "", fmt.Errorf("backup source URL not found in %s", sqlPath)
	}

	return validateURL(backupURL)
}

// BackupTablePrefix reads the original WordPress table prefix from SQL backup
// metadata, falling back to the default prefix when the dump does not expose
// one.
//
// Parameters:
// - sqlPath: SQL dump path to inspect
//
// Returns:
// - the detected or default table prefix
// - an error when the file cannot be read or metadata is invalid
func BackupTablePrefix(sqlPath string) (string, error) {
	file, err := os.Open(sqlPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	var tablePrefix string
	parseLine := func(line string) error {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "# Table prefix:"):
			value, err := normalizeTablePrefix(strings.TrimSpace(strings.TrimPrefix(line, "# Table prefix:")))
			if err != nil {
				return err
			}
			tablePrefix = value
			return errStopReading
		case strings.HasPrefix(line, "# Table: `"), strings.HasPrefix(line, "CREATE TABLE `"), strings.HasPrefix(line, "DROP TABLE IF EXISTS `"):
			tableName := tableNameFromLine(line)
			if prefix, ok := tablePrefixFromTableName(tableName); ok {
				tablePrefix = prefix
				return errStopReading
			}
		}
		return nil
	}

	err = scanMetadataHeader(file, parseLine)
	if err != nil && !errors.Is(err, errStopReading) {
		return "", err
	}
	if tablePrefix != "" {
		return tablePrefix, nil
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	err = forEachLine(file, parseLine)
	if err != nil && !errors.Is(err, errStopReading) {
		return "", err
	}
	if tablePrefix != "" {
		return tablePrefix, nil
	}

	return DefaultTablePrefix, nil
}

// normalizeTablePrefix trims and validates a WordPress table prefix.
//
// Parameters:
// - prefix: raw table prefix value
//
// Returns:
// - the normalized prefix
// - an error when the prefix is empty or contains invalid characters
func normalizeTablePrefix(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", fmt.Errorf("table prefix is empty")
	}
	for _, r := range prefix {
		if r != '_' && (r < '0' || r > '9') && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return "", fmt.Errorf("invalid table prefix: %s", prefix)
		}
	}

	return prefix, nil
}

// tableNameFromLine extracts a SQL table name from supported dump lines.
//
// Parameters:
// - line: SQL dump line to inspect
//
// Returns:
// - the table name, or an empty string when none can be parsed
func tableNameFromLine(line string) string {
	for _, marker := range []string{"# Table: `", "CREATE TABLE `", "DROP TABLE IF EXISTS `"} {
		if !strings.HasPrefix(line, marker) {
			continue
		}
		name := strings.TrimPrefix(line, marker)
		end := strings.IndexByte(name, '`')
		if end <= 0 {
			return ""
		}
		return name[:end]
	}

	return ""
}

// tablePrefixFromTableName infers a WordPress table prefix from a table name.
//
// Parameters:
// - tableName: database table name to inspect
//
// Returns:
// - the inferred table prefix
// - true when the table name matches a known WordPress table suffix
func tablePrefixFromTableName(tableName string) (string, bool) {
	for _, suffix := range []string{
		"options",
		"users",
		"usermeta",
		"posts",
		"postmeta",
		"terms",
		"term_taxonomy",
		"term_relationships",
		"termmeta",
		"commentmeta",
		"comments",
		"links",
	} {
		if !strings.HasSuffix(tableName, suffix) || len(tableName) <= len(suffix) {
			continue
		}

		prefix, err := normalizeTablePrefix(strings.TrimSuffix(tableName, suffix))
		if err == nil {
			return prefix, true
		}
	}

	return "", false
}

var errStopReading = errors.New("stop reading")

// scanMetadataHeader scans the bounded SQL dump header for metadata lines.
//
// Parameters:
// - file: SQL dump file positioned at the beginning
// - fn: callback invoked for each line
//
// Returns:
// - an error returned by the reader or callback
//
// Side effects:
// - advances the file read position
func scanMetadataHeader(file *os.File, fn func(line string) error) error {
	return forEachLine(io.LimitReader(file, metadataHeaderLimit), fn)
}

// forEachLine streams lines from reader without imposing scanner token limits.
//
// Parameters:
// - reader: source stream to read
// - fn: callback invoked for each line
//
// Returns:
// - an error returned by the reader or callback
func forEachLine(reader io.Reader, fn func(line string) error) error {
	buf := make([]byte, 0, 64*1024)
	lineBuf := bytes.NewBuffer(buf)
	chunk := make([]byte, 32*1024)

	for {
		n, err := reader.Read(chunk)
		if n > 0 {
			start := 0
			for i := 0; i < n; i++ {
				if chunk[i] != '\n' {
					continue
				}

				lineBuf.Write(chunk[start:i])
				line := strings.TrimSuffix(lineBuf.String(), "\r")
				lineBuf.Reset()
				if callErr := fn(line); callErr != nil {
					return callErr
				}
				start = i + 1
			}
			if start < n {
				lineBuf.Write(chunk[start:n])
			}
		}

		if err != nil {
			if err == io.EOF {
				if lineBuf.Len() == 0 {
					return nil
				}
				return fn(strings.TrimSuffix(lineBuf.String(), "\r"))
			}
			return err
		}
	}
}

// validateURL parses raw and rejects values that do not contain both scheme
// and host parts.
//
// Parameters:
// - raw: raw URL string to validate
//
// Returns:
// - the normalized URL string
// - an error when the URL is missing a scheme or host
func validateURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid backup URL: %s", raw)
	}

	return parsed.String(), nil
}
