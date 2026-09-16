package registry

import (
	"context"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Registry struct {
	Repo name.Repository
	Auth authn.Authenticator
}

func NewRegistry(repository string) (*Registry, error) {
	repo, err := name.NewRepository(repository, name.StrictValidation)
	if err != nil {
		return nil, err
	}
	// use the docker config file credentials
	auth, err := authn.DefaultKeychain.Resolve(repo.Registry)
	if err != nil {
		return nil, err
	}
	return &Registry{
		Repo: repo,
		Auth: auth,
	}, nil
}

func (r *Registry) GetAllTags(ctx context.Context, repository string) ([]string, error) {
	tags, err := remote.List(r.Repo, remote.WithAuth(r.Auth), remote.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// VerifyPull reports whether the given tag resolves to a pullable image for the
// requested platform. It fetches the manifest (and, for a manifest list, the
// matching platform entry) without downloading any layer blobs. A non-nil error
// means the tag is currently unresolvable for that platform — for example a
// partial or in-flight push where the tag is listed but its manifest is broken.
func (r *Registry) VerifyPull(ctx context.Context, tag string, platform v1.Platform) error {
	ref := r.Repo.Tag(tag)
	img, err := remote.Image(
		ref,
		remote.WithAuth(r.Auth),
		remote.WithPlatform(platform),
		remote.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	// Force the network round-trip that resolves the reference (and the
	// matching platform entry if it is an index). This only fetches manifests
	// and config, never the layer blobs.
	_, err = img.Digest()
	return err
}
