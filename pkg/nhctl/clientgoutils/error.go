package clientgoutils

type ErrorType string

const (
	InvalidYaml ErrorType = "InvalidYaml"
)

type TypedError struct {
	error
	ErrorType ErrorType
	Mes       string
}
