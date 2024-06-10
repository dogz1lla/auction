package users

type User struct {
	Name    string
	IsAdmin bool
}

type Users []*User

func NewUser(name string) *User {
	return &User{Name: name, IsAdmin: name == "admin"}
}
