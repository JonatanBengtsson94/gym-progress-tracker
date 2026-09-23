// Command createuser adds a user to the database. There is intentionally no
// registration endpoint, so this is the way to create accounts.
//
// The password is read from stdin: prompted for twice without echo when stdin
// is a terminal, otherwise read from the first line of input. Only its hash
// is stored.
//
//	go run ./cmd/createuser -username alice -first-name Alice -last-name Anderson
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/config"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/database"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
)

type userCreator interface {
	CreateUser(ctx context.Context, username string, plainPassword string, firstName string, lastName string) (user.User, error)
}

// passwordReader returns the password the user typed.
type passwordReader func() (string, error)

func main() {
	cfg, err := config.LoadPostgresConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg)
	if err != nil {
		slog.Error("Failed to setup database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	service := user.NewUserService(user.NewPostgresUserRepository(pool))

	readPassword := func() (string, error) { return readLine(os.Stdin) }
	if term.IsTerminal(int(os.Stdin.Fd())) {
		readPassword = func() (string, error) { return promptPassword(int(os.Stdin.Fd()), os.Stderr) }
	}

	if err := run(ctx, os.Args[1:], readPassword, os.Stdout, service); err != nil {
		fmt.Fprintln(os.Stderr, "createuser:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, readPassword passwordReader, stdout io.Writer, service userCreator) error {
	flags := flag.NewFlagSet("createuser", flag.ContinueOnError)
	username := flags.String("username", "", "username used to log in (required)")
	firstName := flags.String("first-name", "", "first name (required)")
	lastName := flags.String("last-name", "", "last name (required)")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *username == "" || *firstName == "" || *lastName == "" {
		flags.Usage()
		return errors.New("-username, -first-name and -last-name are required")
	}

	plainPassword, err := readPassword()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	created, err := service.CreateUser(ctx, *username, plainPassword, *firstName, *lastName)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Created user %q with id %d\n", created.UserName, created.UserId)
	return nil
}

func readLine(r io.Reader) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && line != "") {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func promptPassword(fd int, prompt io.Writer) (string, error) {
	fmt.Fprint(prompt, "Password: ")
	first, err := term.ReadPassword(fd)
	fmt.Fprintln(prompt)
	if err != nil {
		return "", err
	}

	fmt.Fprint(prompt, "Repeat password: ")
	second, err := term.ReadPassword(fd)
	fmt.Fprintln(prompt)
	if err != nil {
		return "", err
	}

	if string(first) != string(second) {
		return "", errors.New("passwords do not match")
	}

	return string(first), nil
}
