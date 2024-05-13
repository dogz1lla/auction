package templating

type LoginPage struct {
	Form FormData
}

func NewLoginPage() LoginPage {
	return LoginPage{Form: NewFormData()}
}
