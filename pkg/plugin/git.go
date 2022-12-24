package plugin

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/jrcichra/image-gatherer/pkg/registry"
)

type Git struct {
	State
}

var _ InputPlugin = &Git{}

func getTempDir() (string, error) {
	return os.MkdirTemp("", "")
}

func buildAuth(username, password, sshStr string) (transport.AuthMethod, error) {
	var auth transport.AuthMethod
	if username != "" && password != "" {
		auth = &http.BasicAuth{
			Username: username,
			Password: password,
		}
	} else if sshStr == "true" {
		user, err := user.Current()
		if err != nil {
			return auth, err
		}
		// TODO make SSH more configurable
		homedir := fmt.Sprintf("%s/.ssh/id_rsa", user.HomeDir)
		auth, err = ssh.NewPublicKeysFromFile("git", homedir, "")
		if err != nil {
			return auth, err
		}
	}
	return auth, nil
}

func (g *Git) GetTag(ctx context.Context, container string, options map[string]string) (string, error) {
	username, _ := os.LookupEnv(options["username_env"])
	password, _ := os.LookupEnv(options["password_env"])
	ssh := options["ssh"]
	branch := options["branch"]
	url := options["url"]
	tempDir, err := getTempDir()
	defer func() {
		err = os.RemoveAll(tempDir)
		if err != nil {
			panic(err)
		}
	}()
	if err != nil {
		return "", err
	}
	fullBranchName := fmt.Sprintf("refs/heads/%s", branch)
	auth, err := buildAuth(username, password, ssh)
	if err != nil {
		return "", err
	}
	_, err = git.PlainCloneContext(ctx, tempDir, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.ReferenceName(fullBranchName),
		SingleBranch:  true,
		Auth:          auth,
	})
	if err != nil {
		return "", err
	}
	repo, err := git.PlainOpen(tempDir)
	if err != nil {
		return "", err
	}

	ref, err := repo.Reference(plumbing.ReferenceName(fullBranchName), true)
	if err != nil {
		return "", err
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", err
	}
	if commit == nil {
		return "", fmt.Errorf("commit returned was nil")
	}
	fullHash := commit.Hash.String()
	r, err := registry.NewRegistry(container)
	if err != nil {
		return "", err
	}
	tags, err := r.GetAllTags(ctx, container)
	if err != nil {
		return "", err
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
	// digest, err := r.GetDigestFromTag(ctx, container, matchedTag)
	// if err != nil {
	// 	return "", err
	// }
	// the response should be the full output
	result := fmt.Sprintf("%s:%s", container, matchedTag)
	return result, nil
}

var _ OutputPlugin = &Git{}

func (g *Git) Synth(ctx context.Context, options map[string]string) error {
	filename := options["filename"]
	username, _ := os.LookupEnv(options["username_env"])
	password, _ := os.LookupEnv(options["password_env"])
	ssh := options["ssh"]
	branch := options["branch"]
	url := options["url"]
	tempDir, err := getTempDir()
	defer func() {
		err = os.RemoveAll(tempDir)
		if err != nil {
			panic(err)
		}
	}()

	auth, err := buildAuth(username, password, ssh)
	if err != nil {
		return err
	}
	fullBranchName := fmt.Sprintf("refs/heads/%s", branch)
	repo, err := git.PlainCloneContext(ctx, tempDir, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.ReferenceName(fullBranchName),
		SingleBranch:  true,
		Auth:          auth,
	})
	if err != nil {
		return err
	}
	fullPath := filepath.Join(tempDir, filename)
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
	w, err := repo.Worktree()
	if err != nil {
		return err
	}
	status, err := w.Status()
	if err != nil {
		return err
	}
	// do nothing if there's no change
	if status.IsClean() {
		return nil
	}
	if _, err := w.Add(filename); err != nil {
		return err
	}
	message := fmt.Sprintf("chore: update %s", filename)
	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Image Gatherer",
			Email: "imagegatherer@jrcichra.dev",
			When:  time.Now(),
		},
		AllowEmptyCommits: false,
	})
	if err != nil {
		return err
	}
	err = repo.PushContext(ctx, &git.PushOptions{
		Auth: auth,
	})
	if err != nil {
		return err
	}
	return nil
}
