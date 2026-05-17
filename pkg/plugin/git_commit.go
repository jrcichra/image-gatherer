package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/jrcichra/image-gatherer/pkg/registry"
)

type GitCommit struct{}

var _ InputPlugin = &GitCommit{}

func (g *GitCommit) GetTag(ctx context.Context, container string, options map[string]string) (string, error) {
	url := options["url"]
	branch := options["branch"]
	username, _ := os.LookupEnv(options["username_env"])
	password, _ := os.LookupEnv(options["password_env"])

	auth, err := buildAuth(username, password, options["ssh"], options["ssh_key_path"])
	if err != nil {
		return "", err
	}

	r, err := registry.NewRegistry(container)
	if err != nil {
		return "", err
	}
	tags, err := r.GetAllTags(ctx, container)
	if err != nil {
		return "", err
	}

	tempDir, err := getTempDir()
	if err != nil {
		return "", err
	}
	defer removeTempDir(tempDir)

	fullBranchName := fmt.Sprintf("refs/heads/%s", branch)
	repo, err := git.PlainCloneContext(ctx, tempDir, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.ReferenceName(fullBranchName),
		SingleBranch:  true,
		Auth:          auth,
	})
	if err != nil {
		return "", err
	}
	ref, err := repo.Reference(plumbing.ReferenceName(fullBranchName), true)
	if err != nil {
		return "", err
	}

	commit, err := repo.CommitObject(ref.Hash())
	var matchedTag string
	for matchedTag == "" {
		if err != nil {
			return "", err
		}
		if commit == nil {
			return "", fmt.Errorf("commit is nil")
		}
		fullHash := commit.Hash.String()
		matchedTag = findMatchingTag(fullHash, tags)
		if matchedTag == "" {
			slog.Info("no image for commit, checking parent", "container", container, "hash", fullHash)
			commit, err = commit.Parents().Next()
			if err != nil {
				return "", fmt.Errorf("no matching container image found in commit history: %w", err)
			}
		}
	}

	return fmt.Sprintf("%s:%s", container, matchedTag), nil
}
