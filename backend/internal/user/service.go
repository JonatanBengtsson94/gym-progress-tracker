package user

import (
	"context"
	"strings"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/password"
)

type UserRepository interface {
	GetUserByUserId(context.Context, uint32) (User, error)
	CreateUser(context.Context, User) (User, error)
}

type UserServiceImpl struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserServiceImpl {
	return &UserServiceImpl{repo: repo}
}

func (s *UserServiceImpl) GetUser(ctx context.Context, userId uint32) (User, error) {
	return s.repo.GetUserByUserId(ctx, userId)
}

// CreateUser hashes plainPassword and stores the user. The plain text
// password is never passed on to the repository.
func (s *UserServiceImpl) CreateUser(ctx context.Context, username string, plainPassword string, firstName string, lastName string) (User, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(firstName) == "" || strings.TrimSpace(lastName) == "" {
		return User{}, ErrUserFieldsRequired
	}

	hash, err := password.Hash(plainPassword)
	if err != nil {
		return User{}, err
	}

	return s.repo.CreateUser(ctx, User{
		UserName:     username,
		PasswordHash: hash,
		FirstName:    firstName,
		LastName:     lastName,
	})
}
