package user

import "context"

type UserRepository interface {
	GetUserByUserId(context.Context, uint32) (User, error)
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
