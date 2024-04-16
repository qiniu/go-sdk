package errors

import "fmt"

type (
	MissingRequiredFieldError struct {
		Name string
	}
)

func (err MissingRequiredFieldError) Error() string {
	return fmt.Sprintf("missing required field `%s`", err.Name)
}
