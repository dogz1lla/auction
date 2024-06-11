package users

import (
	"errors"
	"fmt"
)

type User struct {
	Name    string
	IsAdmin bool
}

type Users []*User

func NewUser(name string) *User {
	return &User{Name: name, IsAdmin: name == "admin"}
}

func (users Users) GetUser(userName string) (*User, error) {
	for _, user := range users {
		if user.Name == userName {
			return user, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("user %s not found", userName))
}

// // maps websocket client uids into actual user names
// type ClientToUserMap map[string]string
