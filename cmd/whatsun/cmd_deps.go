package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/upsun/whatsun/internal/fsgitignore"
	"github.com/upsun/whatsun/pkg/dep"
)

func depsCmd() *cobra.Command {
	var ignore []string
	var includeIndirect bool
	var includeDev bool
	var plain bool
	cmd := &cobra.Command{
		Use:   "deps [path]",
		Short: "List dependencies found in the repository",
		Args:  cobra.RangeArgs(0, 1),
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			return runDeps(cmd.Context(), path, ignore, includeIndirect, includeDev, plain, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().StringSliceVar(&ignore, "ignore", []string{},
		"Paths (or patterns) to ignore, adding to defaults.")
	cmd.Flags().BoolVar(&includeIndirect, "include-indirect", false,
		"Include indirect/transitive dependencies in addition to direct dependencies.")
	cmd.Flags().BoolVar(&includeDev, "include-dev", false,
		"Include development-only dependencies.")
	cmd.Flags().BoolVar(&plain, "plain", false,
		"Output plain tab-separated values with header row.")

	return cmd
}

func runDeps(
	ctx context.Context,
	path string,
	ignore []string,
	includeIndirect, includeDev, plain bool,
	stdout, stderr io.Writer,
) error {
	fsys, disableGitIgnore, err := setupFileSystem(ctx, path, stderr)
	if err != nil {
		return err
	}

	dependencies, err := collectAllDependencies(ctx, fsys, ignore, disableGitIgnore)
	if err != nil {
		return fmt.Errorf("failed to collect dependencies: %w", err)
	}

	// Filter dependencies based on flags
	var filteredDeps []dependencyInfo
	for _, depInfo := range dependencies {
		// Skip indirect dependencies if not requested
		if !includeIndirect && !depInfo.Dependency.IsDirect {
			continue
		}
		// Skip dev dependencies unless --include-dev is specified
		if !includeDev && depInfo.Dependency.IsDevOnly {
			continue
		}
		filteredDeps = append(filteredDeps, depInfo)
	}

	if len(filteredDeps) == 0 {
		if !includeIndirect {
			fmt.Fprintln(stderr, "No direct dependencies found.")
		} else {
			fmt.Fprintln(stderr, "No dependencies found.")
		}
		return nil
	}

	if plain {
		outputDepsPlain(filteredDeps, stdout)
	} else {
		tbl := table.NewWriter()
		tbl.AppendHeader(table.Row{"Path", "Tool", "Name", "Constraint", "Version"})

		for _, depInfo := range filteredDeps {
			tbl.AppendRow(table.Row{
				depInfo.Path,
				depInfo.Dependency.ToolName,
				depInfo.Dependency.Name,
				depInfo.Dependency.Constraint,
				depInfo.Dependency.Version,
			})
		}

		fmt.Fprintln(stdout, tbl.Render())
	}
	return nil
}

type dependencyInfo struct {
	Path       string
	Manager    string
	Dependency dep.Dependency
}

func collectAllDependencies(
	ctx context.Context,
	fsys fs.FS,
	ignore []string,
	disableGitIgnore bool,
) ([]dependencyInfo, error) {
	var allDeps []dependencyInfo

	var ignorePatterns = fsgitignore.GetDefaultIgnorePatterns()
	if len(ignore) > 0 {
		ignorePatterns = append(ignorePatterns, fsgitignore.ParsePatterns(ignore, fsgitignore.Split("."))...)
	}

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		// Hard-limit the directory depth to 16.
		if strings.Count(path, string(os.PathSeparator)) >= 16 {
			return filepath.SkipDir
		}

		// Skip .git and node_modules
		if d.Name() == ".git" || d.Name() == "node_modules" {
			return fs.SkipDir
		}

		// Apply gitignore patterns
		if gitignore.NewMatcher(ignorePatterns).Match(fsgitignore.Split(path), true) {
			return fs.SkipDir
		}

		// Parse additional .gitignore files if not disabled
		if !disableGitIgnore {
			patterns, err := fsgitignore.ParseIgnoreFiles(fsys, path)
			if err != nil {
				return err
			}
			ignorePatterns = append(ignorePatterns, patterns...)
		}

		// Try each manager type for this directory
		for _, managerType := range dep.AllManagerTypes {
			manager, err := dep.GetManager(managerType, fsys, path)
			if err != nil {
				return err
			}

			if err := manager.Init(); err != nil {
				return err
			}

			// Get all dependencies by using a wildcard pattern
			deps := manager.Find("*")
			slices.SortStableFunc(deps, func(a, b dep.Dependency) int {
				return strings.Compare(a.Name, b.Name)
			})
			for _, dependency := range deps {
				allDeps = append(allDeps, dependencyInfo{
					Path:       path,
					Manager:    managerType,
					Dependency: dependency,
				})
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return allDeps, nil
}
