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

	"github.com/upsun/whatsun/pkg/files"
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

func setupFileSystem(ctx context.Context, path string, stderr io.Writer) (fsys fs.FS, fromClone bool, err error) {
	note := noter(stderr)

	if files.IsLocal(path) {
		note("Processing local path: %s", path)
		fsys, err = files.LocalFS(path)
		if err != nil {
			return
		}
	} else {
		preClone := time.Now()
		note("Cloning remote repository (into memory): %s", path)
		fsys, err = files.Clone(ctx, path, "")
		if err != nil {
			return
		}
		note("Cloned repository in %v", time.Since(preClone).Truncate(time.Millisecond))
		fromClone = true
	}

	return
}
