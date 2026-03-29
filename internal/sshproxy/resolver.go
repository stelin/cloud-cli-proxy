package sshproxy

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net"

	"github.com/zanel1u/cloud-cli-proxy/internal/network"
	"github.com/zanel1u/cloud-cli-proxy/internal/store/repository"
)

type resolverRepo interface {
	GetUserByShortID(ctx context.Context, shortID string) (repository.User, error)
	GetPrimaryHostByUserID(ctx context.Context, userID string) (repository.Host, error)
}

// RepoResolver implements ContainerResolver using the database repository.
// It validates the user's entry_password, checks that the user is active
// and has a running host, then derives the management network SSH address.
type RepoResolver struct {
	repo resolverRepo
}

func NewRepoResolver(repo resolverRepo) *RepoResolver {
	return &RepoResolver{repo: repo}
}

func (r *RepoResolver) ResolveContainer(ctx context.Context, shortID, password string) (string, error) {
	user, err := r.repo.GetUserByShortID(ctx, shortID)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}

	if user.Status != "active" {
		return "", fmt.Errorf("user suspended")
	}

	if user.EntryPassword == "" || subtle.ConstantTimeCompare([]byte(user.EntryPassword), []byte(password)) != 1 {
		return "", fmt.Errorf("invalid credentials")
	}

	host, err := r.repo.GetPrimaryHostByUserID(ctx, user.ID)
	if err != nil {
		return "", fmt.Errorf("no host available")
	}

	if host.Status != "running" {
		return "", fmt.Errorf("host not running (status: %s)", host.Status)
	}

	access := network.DeriveManagementSSHAccess(host.ID)
	return net.JoinHostPort(access.Host, fmt.Sprintf("%d", access.Port)), nil
}
