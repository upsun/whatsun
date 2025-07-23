package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/upsun/whatsun/pkg/files"
)

func treeCmd() *cobra.Command {
	var ignore []string
	var cmd = &cobra.Command{
		Use:   "tree [path]",
		Short: "Only output a file tree",
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
			return runTree(cmd.Context(), path, ignore, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().StringSliceVar(&ignore, "ignore", []string{},
		"Paths (or patterns) to ignore, adding to defaults.")
	return cmd
}

func runTree(ctx context.Context, path string, ignore []string, stdout, stderr io.Writer) error {
	fsys, disableGitIgnore, err := setupFileSystem(ctx, path, stderr)
	if err != nil {
		return err
	}

	cnf := files.MinimalTreeConfig
	cnf.IgnoreDirs = ignore
	cnf.DisableGitIgnore = disableGitIgnore
	result, err := files.GetTree(fsys, cnf)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, strings.Join(result, "\n"))
	return nil
}
