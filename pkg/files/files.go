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

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"

	"github.com/upsun/whatsun/pkg/fsgitignore"
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
		files[i].Content = RemoveComments(files[i].Name, files[i].Content)
		files[i].Content = ReplaceSecrets(files[i].Content, "[REDACTED]")
		files[i].Cleaned = true
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
		f, err := fsys.Open(name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
				continue
			}
			return nil, fmt.Errorf("failed opening file %s: %w", name, err)
		}
		fi, err := f.Stat()
		if err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("failed to stat file %s: %w", name, err)
		}
		if ignoreMatcher.Match(fsgitignore.Split(name), fi.IsDir()) {
			continue
		}
		// Read up to maxLength+1 to detect truncation
		buf := make([]byte, maxLength+1)
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			_ = f.Close()
			return nil, fmt.Errorf("failed reading file %s: %w", name, err)
		}
		_ = f.Close()
		truncated := n > maxLength
		if truncated {
			n = maxLength
		}
		contents = append(contents, FileData{
			Name:      name,
			Content:   string(buf[:n]),
			Size:      fi.Size(),
			Truncated: truncated,
		})
	}

	slices.SortFunc(contents, func(a, b FileData) int {
		return strings.Compare(a.Name, b.Name)
	})

	return contents, nil
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
