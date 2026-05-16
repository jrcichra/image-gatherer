package plugin

import (
	"context"
	"fmt"
	"log/slog"
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

func removeTempDir(path string) {
	if err := os.RemoveAll(path); err != nil {
		slog.Error("failed to remove temp dir", "path", path, "err", err)
	}
}

func buildAuth(username, password, sshStr, sshKeyPath string) (transport.AuthMethod, error) {
	if username != "" && password != "" {
		return &http.BasicAuth{Username: username, Password: password}, nil
	}
	if sshStr == "true" {
		if sshKeyPath == "" {
			u, err := user.Current()
			if err != nil {
				return nil, err
			}
			sshKeyPath = filepath.Join(u.HomeDir, ".ssh", "id_rsa")
		}
		return ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
	}
	return nil, nil
}

// isHexString reports whether s is a non-empty string of hex digits.
func isHexString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// findMatchingTag looks for a tag whose hex portion (after stripping a "sha-"
// prefix) is a prefix of fullHash. Requires at least 7 hex chars to avoid
// accidental matches.
func findMatchingTag(fullHash string, tags []string) string {
	for _, tag := range tags {
		trimmedTag := strings.TrimPrefix(tag, "sha-")
		if len(trimmedTag) < 7 || !isHexString(trimmedTag) {
			continue
		}
		if strings.HasPrefix(fullHash, trimmedTag) {
			return tag
		}
	}
	return ""
}

func (g *Git) GetTag(ctx context.Context, container string, options map[string]string) (string, error) {
	url := options["url"]
	branch := options["branch"]
	if url == "" {
		return "", fmt.Errorf("git input plugin: url option is required")
	}
	if branch == "" {
		return "", fmt.Errorf("git input plugin: branch option is required")
	}

	username, _ := os.LookupEnv(options["username_env"])
	password, _ := os.LookupEnv(options["password_env"])
	sshKeyPath := options["ssh_key_path"]

	auth, err := buildAuth(username, password, options["ssh"], sshKeyPath)
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

var _ OutputPlugin = &Git{}

func (g *Git) Synth(ctx context.Context, options map[string]string) error {
	url := options["url"]
	branch := options["branch"]
	if url == "" {
		return fmt.Errorf("git output plugin: url option is required")
	}
	if branch == "" {
		return fmt.Errorf("git output plugin: branch option is required")
	}

	filename := options["filename"]
	username, _ := os.LookupEnv(options["username_env"])
	password, _ := os.LookupEnv(options["password_env"])
	sshKeyPath := options["ssh_key_path"]

	authorName := options["commit_author_name"]
	if authorName == "" {
		authorName = "Image Gatherer"
	}
	authorEmail := options["commit_author_email"]
	if authorEmail == "" {
		authorEmail = "imagegatherer@jrcichra.dev"
	}

	auth, err := buildAuth(username, password, options["ssh"], sshKeyPath)
	if err != nil {
		return err
	}

	tempDir, err := getTempDir()
	if err != nil {
		return err
	}
	defer removeTempDir(tempDir)

	fullBranchName := fmt.Sprintf("refs/heads/%s", branch)
	repo, err := git.PlainCloneContext(ctx, tempDir, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.ReferenceName(fullBranchName),
		SingleBranch:  true,
		Auth:          auth,
		Depth:         1,
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
	if status.IsClean() {
		slog.Info("no changes to commit", "filename", filename)
		return nil
	}
	if _, err := w.Add(filename); err != nil {
		return err
	}
	_, err = w.Commit(fmt.Sprintf("chore: update %s", filename), &git.CommitOptions{
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
	return repo.PushContext(ctx, &git.PushOptions{Auth: auth})
}
