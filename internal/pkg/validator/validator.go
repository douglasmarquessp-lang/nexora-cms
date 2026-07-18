package validator

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Validator struct {
	errors []ValidationError
}

func New() *Validator {
	return &Validator{errors: make([]ValidationError, 0)}
}

func (v *Validator) Required(field, value, message string) {
	if strings.TrimSpace(value) == "" {
		v.errors = append(v.errors, ValidationError{Field: field, Message: message})
	}
}

func (v *Validator) MinLength(field, value string, minLen int) {
	if utf8.RuneCountInString(value) < minLen {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("mínimo de %d caracteres", minLen),
		})
	}
}

func (v *Validator) MaxLength(field, value string, maxLen int) {
	if utf8.RuneCountInString(value) > maxLen {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("máximo de %d caracteres", maxLen),
		})
	}
}

func (v *Validator) Email(field, value string) {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	if !emailRegex.MatchString(value) {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: "email inválido",
		})
	}
}

func (v *Validator) Slug(field, value string) {
	slugRegex := regexp.MustCompile(`^[a-z0-9\-]+$`)
	if !slugRegex.MatchString(value) {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: "slug deve conter apenas letras minúsculas, números e hífens",
		})
	}
}

func (v *Validator) Valid() bool {
	return len(v.errors) == 0
}

func (v *Validator) Errors() []ValidationError {
	return v.errors
}
