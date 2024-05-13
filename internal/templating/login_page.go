package templating

type FormData struct {
	Values map[string]string
	Errors map[string]string
}

func NewFormData() FormData {
	return FormData{
		Values: make(map[string]string),
		Errors: make(map[string]string),
	}
}

type LoginPage struct {
	Form FormData
}

func NewLoginPage() LoginPage {
	return LoginPage{Form: NewFormData()}
}
