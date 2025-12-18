package main

import (
	"context"
	"encoding/json"
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/upsun/whatsun/pkg/digest"
)

func digestCmd() *cobra.Command {
	var ignore []string
	var useYAML bool
	var cmd = &cobra.Command{
		Use:   "digest [path]",
		Short: "Output a digest of the repository including the file tree, reports, and the contents of selected files",
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
			return runDigest(cmd.Context(), path, ignore, useYAML, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().StringSliceVar(&ignore, "ignore", []string{},
		"Paths (or patterns) to ignore, adding to defaults.")
	cmd.Flags().BoolVar(&useYAML, "yaml", false,
		"Output in YAML format instead of JSON.")
	return cmd
}

func runDigest(ctx context.Context, path string, ignore []string, useYAML bool, stdout, stderr io.Writer) error {
	fsys, disableGitIgnore, err := setupFileSystem(ctx, path, stderr)
	if err != nil {
		return err
	}

	digestCnf, err := digest.DefaultConfig()
	if err != nil {
		return err
	}
	digestCnf.DisableGitIgnore = disableGitIgnore
	digestCnf.IgnoreFiles = ignore
	digester, err := digest.NewDigester(fsys, digestCnf)
	if err != nil {
		return err
	}
	d, err := digester.GetDigest(ctx)
	if err != nil {
		return err
	}
	if useYAML {
		return yaml.NewEncoder(stdout).Encode(d)
	}
	return json.NewEncoder(stdout).Encode(d)
}
