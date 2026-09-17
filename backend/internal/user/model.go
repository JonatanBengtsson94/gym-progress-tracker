package user

import "errors"

var ErrUserNotFound = errors.New("user not found")

type User struct {
	UserId    uint32
	UserName  string
	Password  string
	FirstName string
	LastName  string
}
