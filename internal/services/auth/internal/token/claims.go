package token

// Type represents the type of token (access or refresh)
type Type string

const (
	TypeAccess  Type = "access"
	TypeRefresh Type = "refresh"
)

// UserClaims holds the claims for a user token
type UserClaims struct {
	Type     Type   `json:"typ"`
	ID       string `json:"id"`
	Email    string `json:"email"`
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
}
