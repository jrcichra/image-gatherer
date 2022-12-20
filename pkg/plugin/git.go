package plugin

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/jrcichra/latest-image-gatherer/pkg/registry"
)

type Git struct {
	Url      string `yaml:"url"`
	Branch   string `yaml:"branch"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	SSH      bool   `yaml:"ssh"`
}

func getTempDir() (string, error) {
	return os.MkdirTemp("", "")
}

func (g *Git) Get(ctx context.Context, container string) (string, string, error) {
	tempDir, err := getTempDir()
	defer func() {
		err = os.RemoveAll(tempDir)
		if err != nil {
			panic(err)
		}
	}()
	if err != nil {
		return "", "", err
	}
	fullBranchName := fmt.Sprintf("refs/heads/%s", g.Branch)

	// check which auth method was used
	var auth transport.AuthMethod
	if g.Username != "" && g.Password != "" {
		auth = &http.BasicAuth{
			Username: g.Username,
			Password: g.Password,
		}
	} else if g.SSH {
		user, err := user.Current()
		if err != nil {
			return "", "", err
		}
		// TODO make SSH more configurable
		homedir := fmt.Sprintf("%s/.ssh/id_rsa", user.HomeDir)
		auth, err = ssh.NewPublicKeysFromFile("git", homedir, "")
		if err != nil {
			return "", "", err
		}
	}

	_, err = git.PlainCloneContext(ctx, tempDir, false, &git.CloneOptions{
		URL:           g.Url,
		ReferenceName: plumbing.ReferenceName(fullBranchName),
		SingleBranch:  true,
		Auth:          auth,
	})
	if err != nil {
		return "", "", err
	}
	repo, err := git.PlainOpen(tempDir)
	if err != nil {
		return "", "", err
	}

	ref, err := repo.Reference(plumbing.ReferenceName(fullBranchName), true)
	if err != nil {
		return "", "", err
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", "", err
	}
	if commit == nil {
		return "", "", fmt.Errorf("commit returned was nil")
	}
	fullHash := commit.Hash.String()
	r, err := registry.NewRegistry(container)
	if err != nil {
		return "", "", err
	}
	tags, err := r.GetAllTags(ctx, container)
	if err != nil {
		return "", "", err
	}
	// loop through the tags and see if the hash is in there
	// TODO improve regex for matching tags to hashes, or expose it
	var matchedTag string
	for _, tag := range tags {
		// clean up the tag
		trimmedTag := strings.TrimPrefix(tag, "sha-")
		// is this tag part of the latest hash?
		if strings.HasPrefix(fullHash, trimmedTag) || strings.HasSuffix(fullHash, trimmedTag) {
			// found it
			matchedTag = tag
			break
		}
	}
	// get the digest for this tag
	digest, err := r.GetDigestFromTag(ctx, container, matchedTag)
	if err != nil {
		return "", "", err
	}
	// the response should be the full output
	result := fmt.Sprintf("%s@%s", container, digest)
	return result, matchedTag, nil
}
