package registry

import (
	"context"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
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
