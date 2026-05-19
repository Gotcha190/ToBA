package wordpress

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	uploadsFallbackBegin = "# BEGIN ToBA Uploads Fallback"
	uploadsFallbackEnd   = "# END ToBA Uploads Fallback"
	wordPressBegin       = "# BEGIN WordPress"
)

// UploadsFallbackBlock builds an Apache rewrite block for skipped uploads.
//
// Parameters:
// - sourceURL: remote WordPress home URL captured from SSH starter data
//
// Returns:
// - the complete fallback block
// - an error when sourceURL cannot be used as a redirect origin
func UploadsFallbackBlock(sourceURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(sourceURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid uploads fallback source URL: %q", sourceURL)
	}

	origin := parsed.Scheme + "://" + parsed.Host
	hostPattern := regexp.QuoteMeta(parsed.Host)

	return strings.Join([]string{
		uploadsFallbackBegin,
		"<IfModule mod_rewrite.c>",
		"RewriteEngine On",
		"",
		"RewriteCond %{HTTP_HOST} !^" + hostPattern + "$ [NC]",
		"RewriteCond %{REQUEST_URI} ^/wp-content/uploads/ [NC]",
		"RewriteCond %{REQUEST_FILENAME} !-f",
		"RewriteCond %{REQUEST_FILENAME} !-d",
		"RewriteRule ^wp-content/uploads/(.*)$ " + origin + "/wp-content/uploads/$1 [R=302,NE,L]",
		"</IfModule>",
		uploadsFallbackEnd,
	}, "\n"), nil
}

// UpsertUploadsFallbackBlock inserts or replaces the ToBA uploads fallback block.
//
// Parameters:
// - existing: existing `.htaccess` content
// - block: generated ToBA fallback block
//
// Returns:
// - updated `.htaccess` content with one ToBA fallback block
func UpsertUploadsFallbackBlock(existing string, block string) string {
	content := removeUploadsFallbackBlock(existing)
	content = strings.TrimLeft(content, "\r\n")
	block = strings.TrimSpace(block)

	if marker := strings.Index(content, wordPressBegin); marker >= 0 {
		prefix := strings.TrimRight(content[:marker], "\r\n")
		suffix := strings.TrimLeft(content[marker:], "\r\n")
		if prefix == "" {
			return block + "\n\n" + suffix
		}
		return prefix + "\n\n" + block + "\n\n" + suffix
	}

	content = strings.TrimRight(content, "\r\n")
	if content == "" {
		return block + "\n"
	}
	return content + "\n\n" + block + "\n"
}

func removeUploadsFallbackBlock(existing string) string {
	content := existing
	for {
		start := strings.Index(content, uploadsFallbackBegin)
		if start < 0 {
			return content
		}

		end := strings.Index(content[start:], uploadsFallbackEnd)
		if end < 0 {
			return content
		}
		end += start + len(uploadsFallbackEnd)

		content = content[:start] + content[end:]
	}
}
