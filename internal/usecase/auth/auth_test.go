package auth

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/entity"
	jwtmanager "mini-instagram/pkg/jwt"
)

type nopLogger struct{}

func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

type fakeUserRepo struct {
	emailExists    bool
	usernameExists bool
	existsErr      error

	byEmail    entity.User
	byEmailErr error

	created    entity.User
	createErr  error
	createCall bool
}

func (f *fakeUserRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	return f.emailExists, f.existsErr
}

func (f *fakeUserRepo) UsernameExists(ctx context.Context, username string) (bool, error) {
	return f.usernameExists, f.existsErr
}

func (f *fakeUserRepo) FindByEmail(ctx context.Context, email string) (entity.User, error) {
	if f.byEmailErr != nil {
		return entity.User{}, f.byEmailErr
	}
	return f.byEmail, nil
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id int64) (entity.User, error) {
	return entity.User{}, entity.ErrNotFound
}

func (f *fakeUserRepo) Create(ctx context.Context, user entity.User) (entity.User, error) {
	f.createCall = true
	if f.createErr != nil {
		return entity.User{}, f.createErr
	}
	user.ID = 1
	f.created = user
	return user, nil
}

func (f *fakeUserRepo) Update(ctx context.Context, user entity.User) error { return nil }

func (f *fakeUserRepo) GetProfileStats(ctx context.Context, userID int64) (int64, int64, int64, error) {
	return 0, 0, 0, nil
}

func (f *fakeUserRepo) IsFollowing(ctx context.Context, followerID, followingID int64) (bool, error) {
	return false, nil
}

func newTestUseCase(repo *fakeUserRepo) *UseCase {
	return New(repo, jwtmanager.New("test-secret"), nopLogger{})
}

func TestSignUp_ValidationError(t *testing.T) {
	repo := &fakeUserRepo{}
	uc := newTestUseCase(repo)

	_, err := uc.SignUp(context.Background(), request.SignUp{
		Email:    "user@example.com",
		FullName: "Jane Doe",
		Username: "ab", // too short
		Password: "correcthorsebattery",
	})

	var vErr *entity.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected *entity.ValidationError, got %T (%v)", err, err)
	}
	if repo.createCall {
		t.Fatal("expected repo.Create not to be called when validation fails")
	}
}

func TestSignUp_EmailTaken(t *testing.T) {
	repo := &fakeUserRepo{emailExists: true}
	uc := newTestUseCase(repo)

	_, err := uc.SignUp(context.Background(), request.SignUp{
		Email:    "user@example.com",
		FullName: "Jane Doe",
		Username: "janedoe",
		Password: "correcthorsebattery",
	})
	if !errors.Is(err, entity.ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestSignUp_UsernameTaken(t *testing.T) {
	repo := &fakeUserRepo{usernameExists: true}
	uc := newTestUseCase(repo)

	_, err := uc.SignUp(context.Background(), request.SignUp{
		Email:    "user@example.com",
		FullName: "Jane Doe",
		Username: "janedoe",
		Password: "correcthorsebattery",
	})
	if !errors.Is(err, entity.ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestSignUp_Success(t *testing.T) {
	repo := &fakeUserRepo{}
	uc := newTestUseCase(repo)

	token, err := uc.SignUp(context.Background(), request.SignUp{
		Email:    "  User@Example.com ",
		FullName: "  Jane Doe ",
		Username: " JaneDoe ",
		Password: "correcthorsebattery",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty access token")
	}
	if repo.created.Email != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", repo.created.Email)
	}
	if repo.created.Username != "janedoe" {
		t.Fatalf("expected normalized username, got %q", repo.created.Username)
	}
	if repo.created.PasswordHash == "" || repo.created.PasswordHash == "correcthorsebattery" {
		t.Fatal("expected password to be hashed")
	}
}

func TestLogin_ValidationError(t *testing.T) {
	repo := &fakeUserRepo{}
	uc := newTestUseCase(repo)

	_, err := uc.Login(context.Background(), request.Login{Email: "", Password: "secret"})

	var vErr *entity.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected *entity.ValidationError, got %T (%v)", err, err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &fakeUserRepo{byEmailErr: entity.ErrNotFound}
	uc := newTestUseCase(repo)

	_, err := uc.Login(context.Background(), request.Login{Email: "user@example.com", Password: "secret"})
	if !errors.Is(err, entity.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	repo := &fakeUserRepo{byEmail: entity.User{ID: 1, Email: "user@example.com", PasswordHash: string(hash), IsActive: false}}
	uc := newTestUseCase(repo)

	_, err := uc.Login(context.Background(), request.Login{Email: "user@example.com", Password: "secret"})
	if !errors.Is(err, entity.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	repo := &fakeUserRepo{byEmail: entity.User{ID: 1, Email: "user@example.com", PasswordHash: string(hash), IsActive: true}}
	uc := newTestUseCase(repo)

	_, err := uc.Login(context.Background(), request.Login{Email: "user@example.com", Password: "wrong"})
	if !errors.Is(err, entity.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	repo := &fakeUserRepo{byEmail: entity.User{ID: 1, Email: "user@example.com", PasswordHash: string(hash), IsActive: true}}
	uc := newTestUseCase(repo)

	token, err := uc.Login(context.Background(), request.Login{Email: "User@Example.com", Password: "secret"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty access token")
	}
}
