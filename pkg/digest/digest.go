package digest

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/upsun/whatsun"
	"github.com/upsun/whatsun/pkg/eval"
	"github.com/upsun/whatsun/pkg/rules"
)

type Config struct {
	DisableGitIgnore bool     // Disable parsing .gitignore files.
	IgnoreFiles      []string // Other "gitignore" file patterns to ignore.
	ReadFiles        []string // Files to read in the project.
	MaxFileLength    int      // Truncate file contents beyond this length.

	Rulesets  []rules.RulesetSpec // Rules to run in each directory.
	ExprCache eval.Cache          // The expression cache.
}

func DefaultConfig() (*Config, error) {
	rulesets, err := whatsun.LoadRulesets()
	if err != nil {
		return nil, err
	}
	exprCache, err := whatsun.LoadExpressionCache()
	if err != nil {
		return nil, err
	}
	return &Config{
		ReadFiles:     defaultReadFiles,
		MaxFileLength: 2048,
		Rulesets:      rulesets,
		ExprCache:     exprCache,
	}, nil
}

type Digester struct {
	fsys     fs.FS
	cnf      *Config
	analyzer *rules.Analyzer
}

func NewDigester(fsys fs.FS, cnf *Config) (*Digester, error) {
	analyzer, err := rules.NewAnalyzer(cnf.Rulesets, &rules.AnalyzerConfig{
		CELExpressionCache: cnf.ExprCache,
		DisableGitIgnore:   cnf.DisableGitIgnore,
		IgnoreDirs:         cnf.IgnoreFiles,
	})
	if err != nil {
		return nil, err
	}
	return &Digester{fsys: fsys, cnf: cnf, analyzer: analyzer}, nil
}

type Digest struct {
	Tree          string              `json:"tree" yaml:"tree"`
	Reports       map[string][]Report `json:"reports" yaml:"reports"` // Grouped by path
	SelectedFiles []FileData          `json:"selected_files" yaml:"selected_files"`
}

var defaultReadFiles = []string{
	"docker-compose.yml",
	"Dockerfile",
	"Makefile",
	"README",
	"README.md",
	"AGENT.md",
	"AGENTS.md",
	"CLAUDE.md",
}

func (d *Digester) GetDigest(_ context.Context) (*Digest, error) {
	tree, err := GetTree(d.fsys, MinimalTreeConfig)
	if err != nil {
		return nil, err
	}

	reports, err := d.analyzer.Analyze(context.Background(), d.fsys, ".")
	if err != nil {
		return nil, err
	}

	var readFiles []string
	readFiles = append(readFiles, d.cnf.ReadFiles...)
	readFiles = append(readFiles, customReadFiles(reports)...)

	fileList, err := ReadMultiple(d.fsys, d.cnf.MaxFileLength, readFiles...)
	if err != nil {
		return nil, err
	}

	return &Digest{
		Tree:          strings.Join(tree, "\n"),
		Reports:       formatReports(reports),
		SelectedFiles: Clean(fileList),
	}, nil
}

// customReadFiles returns project-specific files that should be read (as glob patterns).
func customReadFiles(reports []rules.Report) []string {
	var readFiles []string
	for _, report := range reports {
		for _, f := range append(defaultReadFiles, report.ReadFiles...) {
			readFiles = append(readFiles, filepath.Join(report.Path, f))
		}
	}
	return readFiles
}

// Report is a simpler, more easily serialized, version of rules.Report.
type Report struct {
	Result  string         `json:"result" yaml:"result"`
	With    map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty,flow"`
	Ruleset string         `json:"ruleset" yaml:"ruleset"`
	Groups  []string       `json:"groups,omitempty" yaml:"groups,omitempty,flow"`
}

func formatReports(reports []rules.Report) map[string][]Report {
	var pathReports = make(map[string][]Report)
	for _, report := range reports {
		if report.Maybe {
			continue
		}

		var with = make(map[string]any, len(report.With))
		for k, v := range report.With {
			if str, ok := v.Value.(string); ok && len(str) == 0 {
				continue
			}
			with[k] = v.Value
		}

		pathReports[report.Path] = append(pathReports[report.Path], Report{
			Result:  report.Result,
			Ruleset: report.Ruleset,
			Groups:  report.Groups,
			With:    with,
		})
	}

	return pathReports
}
