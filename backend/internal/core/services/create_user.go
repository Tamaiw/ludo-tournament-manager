// Package services contains the use-case services of the Ludo Tournament Manager.
// Each service takes its dependencies via struct fields and exposes one or more
// Handle(ctx, cmd) methods. No I/O lives in this package; all I/O is via ports.
package services

import (
	"context"
	"errors"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/domain"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/ports"
)

// CreateUser service: hashes via PasswordHasher and persists a User.
// Used by both the seed CLI (T02) and the accept-invite flow (T07).
type CreateUser struct {
	Users   ports.UserRepository
	Hasher  ports.PasswordHasher
	Clock   ports.Clock
}

type CreateUserCmd struct {
	Email    string
	Name     string
	Password string
}

func (c CreateUser) Handle(ctx context.Context, cmd CreateUserCmd) (domain.User, error) {
	if cmd.Password == "" {
		return domain.User{}, errors.New("password must not be empty")
	}
	hash, err := c.Hasher.Hash(cmd.Password)
	if err != nil {
		return domain.User{}, err
	}
	u, err := domain.NewUser(cmd.Email, cmd.Name, hash, c.Clock.Now())
	if err != nil {
		return domain.User{}, err
	}
	if existing, err := c.Users.FindByEmail(ctx, u.Email); err == nil && existing.ID != "" {
		return domain.User{}, domain.ErrUserExists
	}
	if err := c.Users.Save(ctx, u); err != nil {
		return domain.User{}, err
	}
	return u, nil
}