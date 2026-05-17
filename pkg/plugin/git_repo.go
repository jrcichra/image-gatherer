package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"gopkg.in/yaml.v3"
)

type GitRepo struct {
	State
	tempDir  string
	repo     *git.Repository
	auth     transport.AuthMethod
	filename string
}

var _ OutputPlugin = &GitRepo{}

func (g *GitRepo) Open(ctx context.Context, options map[string]string) error {
	url := options["url"]
	branch := options["branch"]
	g.filename = options["filename"]
	username, _ := os.LookupEnv(options["username_env"])
	password, _ := os.LookupEnv(options["password_env"])

	var err error
	g.auth, err = buildAuth(username, password, options["ssh"], options["ssh_key_path"])
	if err != nil {
		return err
	}

	g.tempDir, err = getTempDir()
	if err != nil {
		return err
	}

	fullBranchName := fmt.Sprintf("refs/heads/%s", branch)
	g.repo, err = git.PlainCloneContext(ctx, g.tempDir, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.ReferenceName(fullBranchName),
		SingleBranch:  true,
		Auth:          g.auth,
		Depth:         1,
	})
	if err != nil {
		removeTempDir(g.tempDir)
		return err
	}

	existing, err := os.ReadFile(filepath.Join(g.tempDir, g.filename))
	if err != nil && !os.IsNotExist(err) {
		removeTempDir(g.tempDir)
		return err
	}
	if err == nil {
		if err := yaml.Unmarshal(existing, &g.State); err != nil {
			removeTempDir(g.tempDir)
			return err
		}
	}
	return nil
}

func (g *GitRepo) Close(ctx context.Context, options map[string]string) error {
	defer removeTempDir(g.tempDir)

	authorName := options["commit_author_name"]
	if authorName == "" {
		authorName = "Image Gatherer"
	}
	authorEmail := options["commit_author_email"]
	if authorEmail == "" {
		authorEmail = "imagegatherer@jrcichra.dev"
	}

	fullPath := filepath.Join(g.tempDir, g.filename)
	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	content, err := g.Marshal()
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		return err
	}

	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}
	status, err := w.Status()
	if err != nil {
		return err
	}
	if status.IsClean() {
		slog.Info("no changes to commit", "filename", g.filename)
		return nil
	}
	if _, err := w.Add(g.filename); err != nil {
		return err
	}
	_, err = w.Commit(fmt.Sprintf("chore: update %s", g.filename), &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
		AllowEmptyCommits: false,
	})
	if err != nil {
		return err
	}
	return g.repo.PushContext(ctx, &git.PushOptions{Auth: g.auth})
}
