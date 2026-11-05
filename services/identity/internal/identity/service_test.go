package identity

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Pzquzer/Nocu/shared/auth"
	"github.com/Pzquzer/Nocu/shared/config"
	"github.com/Pzquzer/Nocu/shared/model"
)

type fakeRepo struct {
	users map[string]*model.User
	roles map[string]*model.Role
}

func newFakeRepo() *fakeRepo {
	viewer := &model.Role{Name: model.RoleViewer}
	viewer.ID = uuid.New()
	return &fakeRepo{users: map[string]*model.User{}, roles: map[string]*model.Role{model.RoleViewer: viewer}}
}

func (f *fakeRepo) Create(_ context.Context, u *model.User) error {
	if _, ok := f.users[u.Email]; ok {
		return ErrEmailTaken
	}
	u.ID = uuid.New()
	u.CreatedAt = time.Now()
	f.users[u.Email] = u
	return nil
}

func (f *fakeRepo) FindByEmail(_ context.Context, email string) (*model.User, error) {
	if u, ok := f.users[email]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) FindByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) FindRoleByName(_ context.Context, name string) (*model.Role, error) {
	if r, ok := f.roles[name]; ok {
		return r, nil
	}
	return nil, ErrNotFound
}

func newService(t *testing.T) (*Service, *fakeRepo, *auth.Manager) {
	t.Helper()
	repo := newFakeRepo()
	tokens := auth.NewManager(config.JWT{Secret: "unit-test-secret-value", Issuer: "t", AccessTTL: time.Minute, RefreshTTL: time.Hour})
	svc := NewService(repo, tokens, slog.Default(), Options{DefaultRole: model.RoleViewer, BcryptCost: bcrypt.MinCost})
	return svc, repo, tokens
}

func TestRegisterLoginRefresh(t *testing.T) {
	svc, _, tokens := newService(t)
	ctx := context.Background()

	user, err := svc.Register(ctx, RegisterInput{Email: " Alice@Example.com ", Password: "correct horse", Name: "Alice"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Fatalf("email not normalised: %q", user.Email)
	}
	if got := user.RoleNames(); len(got) != 1 || got[0] != model.RoleViewer {
		t.Fatalf("default role not granted: %v", got)
	}

	pair, err := svc.Login(ctx, "alice@example.com", "correct horse")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	claims, err := tokens.Parse(pair.AccessToken, auth.TokenAccess)
	if err != nil || claims.UserID != user.ID || !claims.HasRole(model.RoleViewer) {
		t.Fatalf("bad access token: %v %+v", err, claims)
	}

	refreshed, err := svc.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if _, err := tokens.Parse(refreshed.AccessToken, auth.TokenAccess); err != nil {
		t.Fatalf("refreshed access token invalid: %v", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, RegisterInput{Email: "not-an-email", Password: "long enough"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for email, got %v", err)
	}
	if _, err := svc.Register(ctx, RegisterInput{Email: "a@b.co", Password: "short"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for password, got %v", err)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()
	in := RegisterInput{Email: "dup@example.com", Password: "long enough"}
	if _, err := svc.Register(ctx, in); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(ctx, in); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestLoginFailures(t *testing.T) {
	svc, repo, _ := newService(t)
	ctx := context.Background()
	user, _ := svc.Register(ctx, RegisterInput{Email: "bob@example.com", Password: "long enough"})

	if _, err := svc.Login(ctx, "bob@example.com", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password: got %v", err)
	}
	if _, err := svc.Login(ctx, "nobody@example.com", "whatever"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown user: got %v", err)
	}

	repo.users[user.Email].IsActive = false
	if _, err := svc.Login(ctx, "bob@example.com", "long enough"); !errors.Is(err, ErrInactive) {
		t.Fatalf("inactive: got %v", err)
	}
}

func TestRefreshRejectsAccessToken(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()
	_, _ = svc.Register(ctx, RegisterInput{Email: "c@example.com", Password: "long enough"})
	pair, _ := svc.Login(ctx, "c@example.com", "long enough")

	if _, err := svc.Refresh(ctx, pair.AccessToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}
