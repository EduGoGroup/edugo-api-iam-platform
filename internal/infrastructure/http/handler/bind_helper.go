package handler

import (
	"errors"
	"fmt"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	sharedErrors "github.com/EduGoGroup/edugo-shared/common/errors"
)

// bindJSON hace ShouldBindJSON extrayendo errores de campo detallados
func bindJSON(c *gin.Context, v interface{}) error {
	if err := c.ShouldBindJSON(v); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			fields := make(map[string]string, len(ve))
			for _, fe := range ve {
				fields[toSnakeCase(fe.Field())] = validationMessage(fe)
			}
			return sharedErrors.NewValidationErrorWithFields("validation failed", fields)
		}
		return sharedErrors.NewValidationError("invalid request body")
	}
	return nil
}

func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	var result []rune
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			hasNext := i+1 < len(runes)
			var next rune
			if hasNext {
				next = runes[i+1]
			}
			if unicode.IsLower(prev) || unicode.IsDigit(prev) ||
				(unicode.IsUpper(prev) && hasNext && unicode.IsLower(next)) {
				result = append(result, '_')
			}
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "field is required"
	case "email":
		return "invalid email format"
	case "min":
		return fmt.Sprintf("minimum length is %s", fe.Param())
	case "max":
		return fmt.Sprintf("maximum length is %s", fe.Param())
	case "uuid":
		return "must be a valid UUID"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	default:
		return fmt.Sprintf("failed validation '%s'", fe.Tag())
	}
}
