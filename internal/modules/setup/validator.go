package setup

import (
	"regexp"
	"strings"
	"unicode"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	if email == "" || len(email) > 254 {
		return false
	}
	return emailRegex.MatchString(email)
}

func isStrongPassword(password string) bool {
	if len(password) < 8 || len(password) > 128 {
		return false
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsNumber(ch):
			hasNumber = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasNumber && hasSpecial
}

func validateInstallRequest(req InstallRequest) error {
	var errs []string

	if req.CmsName == "" {
		errs = append(errs, "cms_name is required")
	}
	if req.AdminName == "" {
		errs = append(errs, "admin_name is required")
	}
	if req.AdminEmail == "" || !isValidEmail(req.AdminEmail) {
		errs = append(errs, "valid admin_email is required")
	}
	if req.Password == "" || !isStrongPassword(req.Password) {
		errs = append(errs, "password must be at least 8 characters with uppercase, lowercase, number, and special character")
	}
	if req.SiteName == "" {
		errs = append(errs, "site_name is required")
	}
	if req.Language == "" {
		errs = append(errs, "language is required")
	}
	if req.Timezone == "" {
		errs = append(errs, "timezone is required")
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}

	return nil
}

type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return "validation failed: " + strings.Join(e.Errors, "; ")
}
