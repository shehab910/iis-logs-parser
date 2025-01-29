package utils

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func InitValidator() {
	Validate = validator.New(validator.WithRequiredStructEnabled())
	Validate.RegisterValidation("strongpassword", passwordValidation)
}

func passwordValidation(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) < 8 {
		return false
	}

	if matched, _ := regexp.MatchString(`[A-Z]`, password); !matched {
		return false
	}

	if matched, _ := regexp.MatchString(`[a-z]`, password); !matched {
		return false
	}

	if matched, _ := regexp.MatchString(`[0-9]`, password); !matched {
		return false
	}

	if matched, _ := regexp.MatchString(`[!@#$%^&*]`, password); !matched {
		return false
	}

	return true
}
