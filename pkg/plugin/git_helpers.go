package plugin

import (
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

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

var hexRe = regexp.MustCompile(`^[0-9a-fA-F]+$`)

// findMatchingTag looks for a tag whose hex portion (after stripping a "sha-"
// prefix) is a prefix of fullHash. Requires at least 7 hex chars to avoid
// accidental matches.
func findMatchingTag(fullHash string, tags []string) string {
	for _, tag := range tags {
		trimmedTag := strings.TrimPrefix(tag, "sha-")
		if len(trimmedTag) < 7 || !hexRe.MatchString(trimmedTag) {
			continue
		}
		if strings.HasPrefix(fullHash, trimmedTag) {
			return tag
		}
	}
	return ""
}
