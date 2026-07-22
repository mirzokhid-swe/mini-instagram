package request

type SignUp struct {
	Email      string
	FullName   string
	Username   string
	Bio        string
	Password   string
	AvatarPath string
}

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
