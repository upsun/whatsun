package files

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/helper/iofs"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"

	"github.com/upsun/whatsun/internal/searchfs"
)

func IsLocal(gitURL string) bool {
	return !strings.Contains(gitURL, "//") && !strings.HasPrefix(gitURL, "git@")
}

func LocalFS(path string) (fs.FS, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return searchfs.New(os.DirFS(abs)), nil
}

// Clone clones a Git repository into an in-memory filesystem.
func Clone(ctx context.Context, gitURL string, refName string) (fs.FS, error) {
	cloneOptions := &git.CloneOptions{
		URL:               gitURL,
		ReferenceName:     plumbing.ReferenceName(refName),
		SingleBranch:      true,
		Depth:             1,
		RecurseSubmodules: 1,
		ShallowSubmodules: true,
	}
	// Use the GITHUB_TOKEN in the environment for HTTPS GitHub URLs.
	if ghToken := os.Getenv("GITHUB_TOKEN"); ghToken != "" && strings.Contains(gitURL, "https://github.com") {
		cloneOptions.Auth = &http.BasicAuth{Username: ghToken}
	}

	gitMemFS := memfs.New()
	_, err := git.CloneContext(ctx, memory.NewStorage(), gitMemFS, cloneOptions)
	if err != nil {
		return nil, err
	}

	return searchfs.New(iofs.New(gitMemFS)), nil
}
