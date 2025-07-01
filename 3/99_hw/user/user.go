package user

type User struct {
	Browser []string `json:"browsers"`
	Email   string   `json:"email"`
	Name    string   `json:"name"`
}
