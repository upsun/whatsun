package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/upsun/whatsun"
	"github.com/upsun/whatsun/pkg/eval"
	"github.com/upsun/whatsun/pkg/files"
	"github.com/upsun/whatsun/pkg/rules"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "whatsun",
		Short: "Analyze a code repository",
	}

	rootCmd.AddCommand(analyzeCmd(), digestCmd(), treeCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(color.RedString(err.Error()))
		os.Exit(1)
	}
}

func noter(stderr io.Writer) func(format string, args ...any) {
	return func(format string, args ...any) {
		fmt.Fprintf(stderr, strings.TrimRight(color.CyanString(format+"\n"), "\n"), args...)
	}
}

func setupFileSystem(ctx context.Context, path string, stderr io.Writer) (fs.FS, bool, error) {
	note := noter(stderr)
	var (
		fsys             fs.FS
		err              error
		disableGitIgnore bool
	)

	if files.IsLocal(path) {
		note("Processing local path: %s", path)
		fsys, err = files.LocalFS(path)
		if err != nil {
			return nil, false, err
		}
	} else {
		preClone := time.Now()
		note("Cloning remote repository (into memory): %s", path)
		fsys, err = files.Clone(ctx, path, "")
		if err != nil {
			return nil, false, err
		}
		note("Cloned repository in %v", time.Since(preClone).Truncate(time.Millisecond))
		disableGitIgnore = true
	}

	return fsys, disableGitIgnore, nil
}

func loadRulesetsAndCache() ([]rules.RulesetSpec, eval.Cache, error) {
	rulesets, err := whatsun.LoadRulesets()
	if err != nil {
		return nil, nil, err
	}
	exprCache, err := whatsun.LoadExpressionCache()
	if err != nil {
		return nil, nil, err
	}
	return rulesets, exprCache, nil
}
