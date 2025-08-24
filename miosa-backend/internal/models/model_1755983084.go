type User struct {
    ID       string `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

type UserRepository interface {
    CreateUser(user *User) error
    GetUserByUsernameAndPassword(username, password string) (*User, error)
    GetUserByID(id string) (*User, error)
}

type JWTToken struct {
    Token string `json:"token"`
}

type AuthService interface {
    Login(username, password string) (*JWTToken, error)
    Register(user *User) (*JWTToken, error)
}

type TokenManager interface {
    GenerateToken(user *User) (*JWTToken, error)
    ValidateToken(token string) (*User, error)
}