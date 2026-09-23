package password_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/password"
)

func TestHash_MatchesOriginalPassword(t *testing.T) {
	hash, err := password.Hash("secret")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	if hash == "secret" {
		t.Fatal("Hash returned the plain text password")
	}
	if !password.Matches(hash, "secret") {
		t.Error("expected hash to match the original password")
	}
}

func TestHash_DoesNotMatchOtherPassword(t *testing.T) {
	hash, err := password.Hash("secret")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	if password.Matches(hash, "wrong-password") {
		t.Error("expected hash not to match a different password")
	}
}

func TestHash_IsSalted(t *testing.T) {
	first, err := password.Hash("secret")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}
	second, err := password.Hash("secret")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	if first == second {
		t.Error("expected two hashes of the same password to differ")
	}
}

func TestHash_EmptyPassword(t *testing.T) {
	_, err := password.Hash("")
	if !errors.Is(err, password.ErrPasswordRequired) {
		t.Fatalf("expected ErrPasswordRequired, got %v", err)
	}
}

func TestHash_TooLongPassword(t *testing.T) {
	_, err := password.Hash(strings.Repeat("a", 73))
	if !errors.Is(err, password.ErrPasswordTooLong) {
		t.Fatalf("expected ErrPasswordTooLong, got %v", err)
	}
}

func TestMatches_PlainTextIsNotAValidHash(t *testing.T) {
	if password.Matches("secret", "secret") {
		t.Error("expected a plain text value stored as a hash never to match")
	}
}
