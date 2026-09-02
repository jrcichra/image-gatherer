package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/jrcichra/image-gatherer/pkg/registry"
)

type GitCommit struct{}

var _ InputPlugin = &GitCommit{}

// startDepth/maxDepth bound the incremental shallow-fetch below: most repos
// match within the first few commits since CI just built HEAD, so we start
// shallow and only pull more history if that guess comes up empty.
const (
	startDepth = 25
	maxDepth   = 1600
)

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

	fullBranchName := fmt.Sprintf("refs/heads/%s", branch)

	var matchedTag string
	for depth := startDepth; matchedTag == "" && depth <= maxDepth; depth *= 4 {
		// A fresh in-memory clone per depth attempt, no working tree and no
		// disk tempdir: only commit metadata is needed to walk history.
		repo, err := git.CloneContext(ctx, memory.NewStorage(), nil, &git.CloneOptions{
			URL:           url,
			ReferenceName: plumbing.ReferenceName(fullBranchName),
			SingleBranch:  true,
			Auth:          auth,
			Depth:         depth,
			NoCheckout:    true,
		})
		if err != nil {
			return "", err
		}
		ref, err := repo.Reference(plumbing.ReferenceName(fullBranchName), true)
		if err != nil {
			return "", err
		}
		commit, err := repo.CommitObject(ref.Hash())
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
				commit, err = commit.Parents().Next()
				if err != nil {
					// Ran out of history at this depth; if we haven't
					// reached the whole repo yet, retry with more.
					slog.Info("no match within depth, widening", "container", container, "depth", depth)
					break
				}
			}
		}
	}
	if matchedTag == "" {
		return "", fmt.Errorf("no matching container image found within %d commits of %s", maxDepth, branch)
	}

	return fmt.Sprintf("%s:%s", container, matchedTag), nil
}
