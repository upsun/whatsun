package analysis

import (
	"context"
	"fmt"
	"io/fs"

	"what"
)

type ProjectAnalyzer struct{}

var _ what.Analyzer = (*ProjectAnalyzer)(nil)

func (*ProjectAnalyzer) GetName() string {
	return "project"
}

func (*ProjectAnalyzer) Analyze(ctx context.Context, fsys fs.FS, root string) (what.Result, error) {
	appAnalyzer := &AppAnalyzer{MaxDepth: 3}
	res, err := appAnalyzer.Analyze(ctx, fsys, root)
	if err != nil {
		return nil, err
	}

	return &Project{AppList: res.(*AppList), Root: root}, nil
}

type Project struct {
	AppList *AppList
	Root    string
}

func (a *Project) GetSummary() string {
	return fmt.Sprintf("Project summary:\n Root: %s\n Apps: %s", a.Root, a.AppList.GetSummary())
}
