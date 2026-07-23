package validation

import (
	"errors"
	"strings"
	"testing"

	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/entity"
)

func asValidationError(t *testing.T, err error) *entity.ValidationError {
	t.Helper()
	var vErr *entity.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected *entity.ValidationError, got %T (%v)", err, err)
	}
	return vErr
}

func TestUsername(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"too short", "ab", true},
		{"too long", strings.Repeat("a", MaxUsername+1), true},
		{"min length ok", strings.Repeat("a", MinUsername), false},
		{"max length ok", strings.Repeat("a", MaxUsername), false},
		{"typical", "john_doe", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Username(tt.value)
			if tt.wantErr {
				vErr := asValidationError(t, err)
				if vErr.Field != "username" {
					t.Fatalf("expected field username, got %q", vErr.Field)
				}
			} else if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestFullName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", true},
		{"too long", strings.Repeat("a", MaxFullName+1), true},
		{"max length ok", strings.Repeat("a", MaxFullName), false},
		{"typical", "Jane Doe", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FullName(tt.value)
			if tt.wantErr {
				vErr := asValidationError(t, err)
				if vErr.Field != "full_name" {
					t.Fatalf("expected field full_name, got %q", vErr.Field)
				}
			} else if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestBio(t *testing.T) {
	if err := Bio(""); err != nil {
		t.Fatalf("expected empty bio to be valid, got %v", err)
	}
	if err := Bio(strings.Repeat("a", MaxBioLength)); err != nil {
		t.Fatalf("expected max-length bio to be valid, got %v", err)
	}

	err := Bio(strings.Repeat("a", MaxBioLength+1))
	vErr := asValidationError(t, err)
	if vErr.Field != "bio" {
		t.Fatalf("expected field bio, got %q", vErr.Field)
	}
}

func TestPassword(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", true},
		{"too short", strings.Repeat("a", MinPassword-1), true},
		{"min length ok", strings.Repeat("a", MinPassword), false},
		{"typical", "correcthorsebattery", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Password(tt.value)
			if tt.wantErr {
				vErr := asValidationError(t, err)
				if vErr.Field != "password" {
					t.Fatalf("expected field password, got %q", vErr.Field)
				}
			} else if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	if err := Email(""); err == nil {
		t.Fatal("expected error for empty email")
	}
	if err := Email("user@example.com"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSignUp(t *testing.T) {
	valid := request.SignUp{
		Email:    "user@example.com",
		FullName: "Jane Doe",
		Username: "janedoe",
		Password: "correcthorsebattery",
		Bio:      "hello world",
	}
	if err := SignUp(valid); err != nil {
		t.Fatalf("expected valid signup to pass, got %v", err)
	}

	tests := []struct {
		name      string
		mutate    func(r *request.SignUp)
		wantField string
	}{
		{"missing email", func(r *request.SignUp) { r.Email = "" }, "email"},
		{"missing full name", func(r *request.SignUp) { r.FullName = "" }, "full_name"},
		{"short username", func(r *request.SignUp) { r.Username = "ab" }, "username"},
		{"short password", func(r *request.SignUp) { r.Password = "short" }, "password"},
		{"long bio", func(r *request.SignUp) { r.Bio = strings.Repeat("a", MaxBioLength+1) }, "bio"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid
			tt.mutate(&req)
			err := SignUp(req)
			vErr := asValidationError(t, err)
			if vErr.Field != tt.wantField {
				t.Fatalf("expected field %q, got %q", tt.wantField, vErr.Field)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	if err := Login(request.Login{Email: "user@example.com", Password: "secret"}); err != nil {
		t.Fatalf("expected valid login to pass, got %v", err)
	}
	if err := Login(request.Login{Email: "", Password: "secret"}); err == nil {
		t.Fatal("expected error for missing email")
	}
	if err := Login(request.Login{Email: "user@example.com", Password: ""}); err == nil {
		t.Fatal("expected error for missing password")
	}
}

func TestProfile(t *testing.T) {
	if err := Profile("janedoe", "Jane Doe", "hello"); err != nil {
		t.Fatalf("expected valid profile to pass, got %v", err)
	}
	if err := Profile("ab", "Jane Doe", "hello"); err == nil {
		t.Fatal("expected error for short username")
	}
	if err := Profile("janedoe", "", "hello"); err == nil {
		t.Fatal("expected error for missing full name")
	}
	if err := Profile("janedoe", "Jane Doe", strings.Repeat("a", MaxBioLength+1)); err == nil {
		t.Fatal("expected error for long bio")
	}
}
