module github.com/gamma-omg/lexi-go/internal/services/image

replace github.com/gamma-omg/lexi-go/internal/pkg => ../../pkg/

go 1.25.1

require (
	github.com/gamma-omg/lexi-go/internal/pkg v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	github.com/magiconair/properties v1.8.10
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
