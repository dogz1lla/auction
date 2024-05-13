package users

type User struct {
	Name string
}

type Users []User

func NewUser(name string) User {
	return User{Name: name}
}
