package files

import (
	"path/filepath"
	"regexp"
	"strings"
)

// RemoveComments attempts to remove comments from supported file types.
func RemoveComments(name, content string) string {
	ext := strings.ToLower(filepath.Ext(name))
	base := strings.ToLower(filepath.Base(name))

	switch {
	case base == "makefile",
		ext == ".yaml", ext == ".yml", ext == ".toml", ext == ".ini", ext == ".hcl",
		ext == ".py":
		return removeHashComments(content)

	case ext == ".json", ext == ".js", ext == ".ts":
		return removeSlashComments(content)

	case ext == ".php":
		return removeHashComments(removeSlashComments(content))

	case ext == ".md", ext == ".html", ext == ".xml":
		return removeHTMLComments(content)

	default:
		return content
	}
}

var (
	hashCommentsPatt           = regexp.MustCompile(`(?m)\s*#.*$`)
	doubleSlashCommentsPatt    = regexp.MustCompile(`(?m)\s*//.*$`)
	slashStarBlockCommentsPatt = regexp.MustCompile(`(?s)/\*.*?\*/`)
	htmlBlockCommentsPatt      = regexp.MustCompile(`(?s)<!--.*?-->`)
)

// removeHashComments removes a # symbol and anything after it in a line.
func removeHashComments(s string) string {
	return hashCommentsPatt.ReplaceAllString(s, "")
}

// removeSlashComments removes comments beginning with '//' and block comments inside '/* */'.
func removeSlashComments(s string) string {
	s = doubleSlashCommentsPatt.ReplaceAllString(s, "")
	return slashStarBlockCommentsPatt.ReplaceAllString(s, "")
}

// removeHTMLComments removes comments inside '<!-- -->'.
func removeHTMLComments(s string) string {
	return htmlBlockCommentsPatt.ReplaceAllString(s, "")
}
