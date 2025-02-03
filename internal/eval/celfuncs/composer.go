package celfuncs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/google/cel-go/cel"
)

// AllComposerFunctions returns CEL functions for reading Composer dependencies in an fs.FS filesystem.
func AllComposerFunctions(fsys *fs.FS, root *string) []cel.EnvOption {
	if root == nil {
		dot := "."
		root = &dot
	}
	return []cel.EnvOption{
		ComposerRequires(fsys, root),
		ComposerLockedVersion(fsys, root),
	}
}

// ComposerRequires defines a CEL function `composer.requires(dep string) -> bool`
// It returns false (without an error) if composer.json does not exist.
func ComposerRequires(fsys *fs.FS, root *string) cel.EnvOption {
	return stringReturnsBoolErr("composer.requires", func(dep string) (bool, error) {
		f, err := (*fsys).Open(filepath.Join(*root, "composer.json"))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return false, nil
			}
			return false, err
		}
		var contents = struct {
			Require map[string]any `json:"require"`
		}{}
		if err := json.NewDecoder(f).Decode(&contents); err != nil {
			return false, err
		}
		if contents.Require == nil {
			return false, nil
		}
		_, ok := contents.Require[dep]
		return ok, nil
	})
}

// ComposerLockedVersion defines a CEL function `composer.lockedVersion(dep string) -> string`
// It returns an empty string if composer.lock or the dependency do not exist.
func ComposerLockedVersion(fsys *fs.FS, root *string) cel.EnvOption {
	return stringReturnsStringErr("composer.lockedVersion", func(dep string) (string, error) {
		f, err := (*fsys).Open(filepath.Join(*root, "composer.lock"))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return "", nil
			}
			return "", err
		}
		var contents = struct {
			Packages []struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"packages"`
		}{}
		if err := json.NewDecoder(f).Decode(&contents); err != nil {
			return "", fmt.Errorf("failed to parse composer.lock: %w", err)
		}
		for _, pkg := range contents.Packages {
			if pkg.Name == dep {
				return pkg.Version, nil
			}
		}
		return "", nil
	})
}
