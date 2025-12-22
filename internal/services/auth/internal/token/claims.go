package token

type Type string

const (
	TypeAccess  Type = "access"
	TypeRefresh Type = "refresh"
)

type UserClaims struct {
	Type     Type   `json:"typ"`
	ID       string `json:"id"`
	Email    string `json:"email"`
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
}
