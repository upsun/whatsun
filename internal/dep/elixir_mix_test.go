package dep_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what/internal/dep"
)

func TestElixir(t *testing.T) {
	fsys := fstest.MapFS{
		"mix.exs": {
			Data: []byte(`
defmodule Phx16.MixProject do
  use Mix.Project

  def project do
    [
      app: :phx16,
      version: "0.1.0",
      elixir: "~> 1.12",
      elixirc_paths: elixirc_paths(Mix.env()),
      compilers: [:gettext] ++ Mix.compilers(),
      start_permanent: Mix.env() == :prod,
      aliases: aliases(),
      deps: deps()
    ]
  end

  defp deps do
    [
      {:phoenix, "~> 1.6.5"},
      {:phoenix_ecto, "~> 4.4"},
      {:ecto_sql, "~> 3.6"},
      {:postgrex, ">= 0.0.0"},
      {:phoenix_html, "~> 3.0"},
      {:phoenix_live_reload, "~> 1.2", runtime: Mix.env() == :dev, only: :dev},
      {:phoenix_live_view, "~> 0.17.5"},
      {:floki, ">= 0.30.0", only: :test},
      {:phoenix_live_dashboard, "~> 0.6"},
      {:esbuild, "~> 0.3", runtime: Mix.env() == :dev},
      {:swoosh, "~> 1.3"},
      {:telemetry_metrics, "~> 0.6"},
      {:telemetry_poller, "~> 1.0"},
      {:gettext, "~> 0.18"},
      {:jason, "~> 1.2"},
      {:plug_cowboy, "~> 2.5"}
    ]
  end
end
			`),
		},
		"mix.lock": {
			Data: []byte(`
{
  "phoenix": {:hex, :phoenix, "1.6.5", "07af307b28a5820b4394f27ac7003df052e065ff651520a58abb16be1eecd519", [:mix], [{:jason, "~> 1.0", [hex: :jason, repo: "hexpm", optional: true]}, {:phoenix_pubsub, "~> 2.0", [hex: :phoenix_pubsub, repo: "hexpm", optional: false]}, {:phoenix_view, "~> 1.0", [hex: :phoenix_view, repo: "hexpm", optional: false]}, {:plug, "~> 1.10", [hex: :plug, repo: "hexpm", optional: false]}, {:plug_cowboy, "~> 2.2", [hex: :plug_cowboy, repo: "hexpm", optional: true]}, {:plug_crypto, "~> 1.2", [hex: :plug_crypto, repo: "hexpm", optional: false]}, {:telemetry, "~> 0.4 or ~> 1.0", [hex: :telemetry, repo: "hexpm", optional: false]}], "hexpm", "97dc3052ca648499280e0636471f1d0439fc623ccdce27d2d8135651421ee80c"}
}
`),
		},
	}

	m, err := dep.GetManager(dep.ManagerTypeElixir, fsys, ".")
	require.NoError(t, err)

	require.NoError(t, m.Init())

	toFind := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"phoenix", []dep.Dependency{{
			Name:       "phoenix",
			Version:    "1.6.5",
			Constraint: "~> 1.6.5",
		}}},
	}
	for _, c := range toFind {
		deps := m.Find(c.pattern)
		assert.Equal(t, c.dependencies, deps)
	}

	toGet := []struct {
		name       string
		dependency dep.Dependency
		found      bool
	}{
		{name: "phoenix", dependency: dep.Dependency{
			Name:       "phoenix",
			Version:    "1.6.5",
			Constraint: "~> 1.6.5",
		}, found: true},
	}
	for _, c := range toGet {
		d, ok := m.Get(c.name)
		assert.Equal(t, c.found, ok)
		assert.Equal(t, c.dependency, d)
	}
}
