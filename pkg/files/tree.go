package files

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"

	"github.com/upsun/whatsun/internal/fsgitignore"
)

// TreeConfig configures the GetTree behavior.
type TreeConfig struct {
	MaxDepth           int     // 0 = unlimited depth
	MaxEntries         int     // 0 = unlimited entries
	MaxEntriesPerLevel float64 // 0.0 = no scaling, otherwise multiplied per level (e.g., 0.5 halves each level)

	// DisableGitIgnore disables handling of .gitignore and .git/info/exclude files.
	//
	// The IgnoreDirs setting will still be respected, and certain directories will
	// always be ignored (namely .git and node_modules). Rules that implement the
	// Ignorer interface will also still be respected.
	DisableGitIgnore bool

	IgnoreDirs []string // Additional directory ignore rules, using git's exclude syntax.
}

// GetTree returns a slice of strings representing the tree structure.
func GetTree(fsys fs.FS, cfg TreeConfig) ([]string, error) {
	var result = []string{"."}

	var ignorePatterns = fsgitignore.GetDefaultIgnorePatterns()
	if len(cfg.IgnoreDirs) > 0 {
		ignorePatterns = append(ignorePatterns, fsgitignore.ParsePatterns(cfg.IgnoreDirs, []string{})...)
	}
	ignorer := gitignore.NewMatcher(ignorePatterns)

	var walk func(currentPath, prefix string, depth int, maxEntries float64) error
	walk = func(currentPath, prefix string, depth int, maxEntries float64) error {
		if cfg.MaxDepth > 0 && depth > cfg.MaxDepth {
			return nil
		}

		entries, err := fs.ReadDir(fsys, currentPath)
		if err != nil {
			return fmt.Errorf("reading directory %q: %w", currentPath, err)
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		var removed int
		if maxEntries > 0 && int(maxEntries) < len(entries) {
			removed += len(entries) - int(maxEntries)
			entries = entries[:int(maxEntries)]
		}

		var displayEntries = make([]fs.DirEntry, 0, len(entries))
		for _, entry := range entries {
			if entry.Name() == ".git" || entry.Name() == "node_modules" {
				continue
			}
			subPath := filepath.Join(currentPath, entry.Name())
			if ignorer.Match(fsgitignore.Split(subPath), entry.IsDir()) {
				continue
			}
			displayEntries = append(displayEntries, entry)
		}

		for i, entry := range displayEntries {
			connector := "├ "
			if i == len(displayEntries)-1 && removed == 0 {
				connector = "└ "
			}

			line := prefix + connector + entry.Name()
			result = append(result, line)

			if entry.IsDir() {
				newPrefix := prefix
				if i == len(displayEntries)-1 && removed == 0 {
					newPrefix += "  "
				} else {
					newPrefix += "│ "
				}
				nextMaxEntries := maxEntries
				if cfg.MaxEntriesPerLevel > 0 {
					nextMaxEntries = maxEntries * cfg.MaxEntriesPerLevel
				}
				subPath := filepath.Join(currentPath, entry.Name())
				if err := walk(subPath, newPrefix, depth+1, nextMaxEntries); err != nil {
					return err
				}
			}
		}

		// Truncated marker
		if removed > 0 {
			line := prefix + "└ " + fmt.Sprintf("... (%d more)", removed)
			result = append(result, line)
		}

		return nil
	}

	startEntries := float64(cfg.MaxEntries)
	err := walk(".", "", 0, startEntries)
	if err != nil {
		return nil, err
	}
	return result, nil
}
