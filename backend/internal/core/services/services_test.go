package services_test

import (
	"context"
	"testing"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/domain"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/ports"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/services"
)

type fakeUserRepo struct {
	users map[string]domain.User
	byID  map[domain.UserID]domain.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[string]domain.User{}, byID: map[domain.UserID]domain.User{}}
}

func (f *fakeUserRepo) Save(_ context.Context, u domain.User) error {
	f.users[u.Email] = u
	f.byID[u.ID] = u
	return nil
}
func (f *fakeUserRepo) FindByID(_ context.Context, id domain.UserID) (domain.User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return domain.User{}, domain.ErrUserNotFound
}
func (f *fakeUserRepo) FindByEmail(_ context.Context, email string) (domain.User, error) {
	if u, ok := f.users[email]; ok {
		return u, nil
	}
	return domain.User{}, domain.ErrUserNotFound
}
func (f *fakeUserRepo) UpdatePasswordHash(_ context.Context, id domain.UserID, hash string) error {
	u, ok := f.byID[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.PasswordHash = hash
	f.byID[id] = u
	return nil
}

// Compile-time assertion that fakeUserRepo satisfies the port.
var _ ports.UserRepository = (*fakeUserRepo)(nil)

func TestCreateUser_HappyPath(t *testing.T) {
	repo := newFakeUserRepo()
	clock := ports.RealClock{}
	hasher := stubHasher{}
	svc := services.CreateUser{Users: repo, Hasher: hasher, Clock: clock}
	u, err := svc.Handle(context.Background(), services.CreateUserCmd{
		Email: "first@example.com", Name: "First", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "first@example.com" {
		t.Errorf("email = %q, want first@example.com", u.Email)
	}
	if u.PasswordHash == "" {
		t.Error("password hash empty")
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo := newFakeUserRepo()
	clock := ports.RealClock{}
	hasher := stubHasher{}
	svc := services.CreateUser{Users: repo, Hasher: hasher, Clock: clock}
	_, err := svc.Handle(context.Background(), services.CreateUserCmd{
		Email: "first@example.com", Name: "First", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	_, err = svc.Handle(context.Background(), services.CreateUserCmd{
		Email: "first@example.com", Name: "Other", Password: "supersecret",
	})
	if err != domain.ErrUserExists {
		t.Errorf("expected ErrUserExists, got %v", err)
	}
}

func TestCreateUser_WeakPassword(t *testing.T) {
	repo := newFakeUserRepo()
	svc := services.CreateUser{Users: repo, Hasher: stubHasher{}, Clock: ports.RealClock{}}
	_, err := svc.Handle(context.Background(), services.CreateUserCmd{
		Email: "x@example.com", Password: "short",
	})
	if err != services.ErrWeakPassword {
		t.Errorf("expected ErrWeakPassword, got %v", err)
	}
}

func TestSignIn_HappyPath(t *testing.T) {
	repo := newFakeUserRepo()
	hasher := stubHasher{}
	svc := services.CreateUser{Users: repo, Hasher: hasher, Clock: ports.RealClock{}}
	_, _ = svc.Handle(context.Background(), services.CreateUserCmd{
		Email: "x@example.com", Name: "X", Password: "supersecret",
	})

	signIn := services.SignIn{Users: repo, Hasher: hasher}
	u, err := signIn.Handle(context.Background(), services.SignInCmd{
		Email: "x@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if u.Email != "x@example.com" {
		t.Errorf("email = %q", u.Email)
	}
}

func TestSignIn_WrongPassword(t *testing.T) {
	repo := newFakeUserRepo()
	hasher := stubHasher{}
	svc := services.CreateUser{Users: repo, Hasher: hasher, Clock: ports.RealClock{}}
	_, _ = svc.Handle(context.Background(), services.CreateUserCmd{
		Email: "x@example.com", Name: "X", Password: "supersecret",
	})
	signIn := services.SignIn{Users: repo, Hasher: hasher}
	_, err := signIn.Handle(context.Background(), services.SignInCmd{
		Email: "x@example.com", Password: "wrong",
	})
	if err != services.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestSignIn_UnknownEmail(t *testing.T) {
	repo := newFakeUserRepo()
	signIn := services.SignIn{Users: repo, Hasher: stubHasher{}}
	_, err := signIn.Handle(context.Background(), services.SignInCmd{
		Email: "ghost@example.com", Password: "supersecret",
	})
	if err != services.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}
