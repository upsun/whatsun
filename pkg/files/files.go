package files

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"

	"github.com/upsun/whatsun/internal/fsgitignore"
)

// FileData wraps file data to pass to a template.
type FileData struct {
	Name      string
	Content   string
	Truncated bool
	Cleaned   bool
	Size      int64
}

// Clean removes comments and redacts secrets in a list of files.
func Clean(files []FileData) []FileData {
	for i := range files {
		orig := files[i].Content
		files[i].Content = RemoveComments(files[i].Name, files[i].Content)
		files[i].Content = ReplaceSecrets(files[i].Content, "[REDACTED]")
		files[i].Content = ReplaceEmails(files[i].Content, "redacted@example.org")
		files[i].Cleaned = orig != files[i].Content
	}
	return files
}

// ReadMultiple reads the given files, specified by glob patterns, up to a limit of bytes each.
// It ignores errors resulting from nonexistence, permissions or empty files.
// It skips any files that are ignored in an ".aiignore" file.
func ReadMultiple(fsys fs.FS, maxLength int, patterns ...string) ([]FileData, error) {
	ignoreMatcher, err := parseAIIgnoreFiles(fsys)
	if err != nil {
		return nil, err
	}

	var globMatches = make(map[string]struct{})
	for _, p := range patterns {
		m, err := fs.Glob(fsys, p)
		if err != nil {
			return nil, err
		}
		for _, match := range m {
			globMatches[match] = struct{}{}
		}
	}

	var contents = make([]FileData, 0, len(globMatches))
	for name := range globMatches {
		fileData, err := readSingleFile(fsys, name, maxLength, ignoreMatcher)
		if err != nil {
			return nil, err
		}
		if fileData != nil {
			contents = append(contents, *fileData)
		}
	}

	slices.SortFunc(contents, func(a, b FileData) int {
		return strings.Compare(a.Name, b.Name)
	})

	return contents, nil
}

// readSingleFile reads a single file, returning nil if the file should be skipped.
func readSingleFile(fsys fs.FS, name string, maxLength int, ignoreMatcher gitignore.Matcher) (*FileData, error) {
	// Check if it's a symbolic link before opening (avoids following the link).
	if statFS, ok := fsys.(fs.StatFS); ok {
		fi, err := statFS.Stat(name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to stat file %s: %w", name, err)
		}

		// Skip directories.
		if fi.IsDir() {
			return nil, nil
		}

		// Skip symbolic links.
		if fi.Mode()&fs.ModeSymlink != 0 {
			return nil, nil
		}
	}

	// Check if the file should be ignored.
	if ignoreMatcher.Match(fsgitignore.Split(name), false) {
		return nil, nil
	}

	f, err := fsys.Open(name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed opening file %s: %w", name, err)
	}
	defer f.Close()

	// Get the total file size for later.
	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", name, err)
	}

	// Read up to maxLength+1 to detect truncation.
	buf := make([]byte, maxLength+1)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed reading file %s: %w", name, err)
	}

	// Skip binary files
	if !isTextContent(buf[:n]) {
		return nil, nil
	}

	truncated := n > maxLength
	if truncated {
		n = maxLength
	}

	return &FileData{
		Name:      name,
		Content:   string(buf[:n]),
		Size:      fi.Size(),
		Truncated: truncated,
	}, nil
}

// isTextContent checks if the given byte slice contains text data.
// It returns false if the content appears to be binary.
func isTextContent(data []byte) bool {
	// Empty files are considered text
	if len(data) == 0 {
		return true
	}

	// Check for null bytes, which are common in binary files
	for _, b := range data {
		if b == 0 {
			return false
		}
	}

	// Check if the content is valid UTF-8
	return utf8.Valid(data)
}

func parseAIIgnoreFiles(fsys fs.FS) (gitignore.Matcher, error) {
	var ps []gitignore.Pattern

	for _, filename := range []string{".aiignore", ".aiexclude"} {
		patterns, err := parseIgnoreFile(fsys, filename)
		if err != nil {
			return nil, err
		}
		ps = append(ps, patterns...)
	}

	// Add global gitignore patterns
	globalPatterns, err := fsgitignore.GetGlobalIgnorePatterns()
	if err == nil && globalPatterns != nil {
		ps = append(ps, globalPatterns...)
	}
	// Note: we silently ignore errors reading global gitignore to avoid breaking file operations

	return gitignore.NewMatcher(ps), nil
}

func parseIgnoreFile(fsys fs.FS, path string) ([]gitignore.Pattern, error) {
	domain := fsgitignore.Split(filepath.Dir(path))
	f, err := fsys.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	return fsgitignore.ParseIgnoreFile(f, domain), nil
}
