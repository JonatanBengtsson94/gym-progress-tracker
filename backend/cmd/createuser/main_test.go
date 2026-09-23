package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/password"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/testutil"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	pool, teardown, err := testutil.StartPostgres(
		ctx,
		filepath.Join("..", "..", "..", "database", "migrations", "*.sql"),
		"",
	)
	if err != nil {
		log.Fatalf("failed to set up test database: %v", err)
	}
	testPool = pool

	code := m.Run()

	teardown()
	os.Exit(code)
}

type mockUserCreator struct {
	createUserFunc func(ctx context.Context, username string, plainPassword string, firstName string, lastName string) (user.User, error)
}

func (m *mockUserCreator) CreateUser(ctx context.Context, username string, plainPassword string, firstName string, lastName string) (user.User, error) {
	return m.createUserFunc(ctx, username, plainPassword, firstName, lastName)
}

func staticPassword(p string) passwordReader {
	return func() (string, error) { return p, nil }
}

func TestRun_CreatesUser(t *testing.T) {
	var gotUsername, gotPassword, gotFirstName, gotLastName string
	service := &mockUserCreator{
		createUserFunc: func(ctx context.Context, username string, plainPassword string, firstName string, lastName string) (user.User, error) {
			gotUsername, gotPassword, gotFirstName, gotLastName = username, plainPassword, firstName, lastName
			return user.User{UserId: 3, UserName: username}, nil
		},
	}

	var stdout bytes.Buffer
	args := []string{"-username", "alice", "-first-name", "Alice", "-last-name", "Anderson"}

	err := run(t.Context(), args, staticPassword("secret"), &stdout, service)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if gotUsername != "alice" || gotPassword != "secret" || gotFirstName != "Alice" || gotLastName != "Anderson" {
		t.Errorf("unexpected CreateUser args: %q %q %q %q", gotUsername, gotPassword, gotFirstName, gotLastName)
	}
	if want := "Created user \"alice\" with id 3\n"; stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRun_MissingFlags(t *testing.T) {
	service := &mockUserCreator{
		createUserFunc: func(ctx context.Context, username string, plainPassword string, firstName string, lastName string) (user.User, error) {
			t.Fatal("CreateUser should not be called when flags are missing")
			return user.User{}, nil
		},
	}

	err := run(t.Context(), []string{"-username", "alice"}, staticPassword("secret"), &bytes.Buffer{}, service)
	if err == nil {
		t.Fatal("expected an error for missing flags")
	}
}

func TestRun_PasswordReadError(t *testing.T) {
	wantErr := errors.New("passwords do not match")
	service := &mockUserCreator{
		createUserFunc: func(ctx context.Context, username string, plainPassword string, firstName string, lastName string) (user.User, error) {
			t.Fatal("CreateUser should not be called when reading the password fails")
			return user.User{}, nil
		},
	}

	readPassword := func() (string, error) { return "", wantErr }
	args := []string{"-username", "alice", "-first-name", "Alice", "-last-name", "Anderson"}

	err := run(t.Context(), args, readPassword, &bytes.Buffer{}, service)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error to wrap %v, got %v", wantErr, err)
	}
}

func TestRun_ServiceError(t *testing.T) {
	service := &mockUserCreator{
		createUserFunc: func(ctx context.Context, username string, plainPassword string, firstName string, lastName string) (user.User, error) {
			return user.User{}, user.ErrUsernameTaken
		},
	}

	args := []string{"-username", "alice", "-first-name", "Alice", "-last-name", "Anderson"}

	err := run(t.Context(), args, staticPassword("secret"), &bytes.Buffer{}, service)
	if !errors.Is(err, user.ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestReadLine(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"newline terminated", "secret\nignored\n", "secret"},
		{"crlf terminated", "secret\r\n", "secret"},
		{"no trailing newline", "secret", "secret"},
		{"keeps surrounding spaces", " secret \n", " secret "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readLine(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("readLine returned error: %v", err)
			}
			if got != tc.want {
				t.Errorf("readLine() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReadLine_EmptyInput(t *testing.T) {
	if _, err := readLine(strings.NewReader("")); err == nil {
		t.Fatal("expected an error for empty input")
	}
}

func TestRun_StoresOnlyPasswordHash(t *testing.T) {
	ctx := t.Context()
	service := user.NewUserService(user.NewPostgresUserRepository(testPool))
	args := []string{"-username", "carol", "-first-name", "Carol", "-last-name", "Clark"}

	if err := run(ctx, args, staticPassword("secret"), &bytes.Buffer{}, service); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var stored string
	err := testPool.QueryRow(ctx, "SELECT password_hash FROM users WHERE username = $1", "carol").Scan(&stored)
	if err != nil {
		t.Fatalf("failed to read stored password: %v", err)
	}

	if strings.Contains(stored, "secret") {
		t.Fatalf("expected only a hash to be stored, got %q", stored)
	}
	if !password.Matches(stored, "secret") {
		t.Error("expected the stored hash to match the password")
	}
}
