package user

import "errors"

var ErrUserNotFound = errors.New("user not found")
var ErrUsernameTaken = errors.New("username is already taken")
var ErrUserFieldsRequired = errors.New("username, first name and last name are required")

type User struct {
	UserId       uint32
	UserName     string
	PasswordHash string
	FirstName    string
	LastName     string
}
