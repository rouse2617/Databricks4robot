package validate

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// allowedStatuses is the set of valid asset statuses.
var allowedStatuses = map[string]bool{
	string(models.AssetStatusApproved):   true,
	string(models.AssetStatusRejected):   true,
	string(models.AssetStatusSuperseded): true,
	string(models.AssetStatusArchived):   true,
}

// RegisterCustomValidators registers custom validation rules with Gin's default validator.
func RegisterCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			label := fld.Tag.Get("label")
			if label != "" {
				return label
			}
			name := fld.Tag.Get("json")
			if name == "" || name == "-" {
				return fld.Name
			}
			if idx := strings.Index(name, ","); idx != -1 {
				name = name[:idx]
			}
			return name
		})
		v.RegisterValidation("valid_status", validateStatus)
		v.RegisterValidation("valid_algo_key", validateAlgoKey)
	}
}

func validateStatus(fl validator.FieldLevel) bool {
	return allowedStatuses[fl.Field().String()]
}

func validateAlgoKey(fl validator.FieldLevel) bool {
	v := fl.Field().String()
	idx := strings.LastIndex(v, "@")
	return idx > 0 && idx < len(v)-1
}

// ValidateStruct validates a struct and returns user-friendly error messages.
func ValidateStruct(obj any) error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return nil
	}
	err := v.Struct(obj)
	if err == nil {
		return nil
	}
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}
	var msgs []string
	for _, fe := range errs {
		msgs = append(msgs, friendlyMessage(fe))
	}
	return fmt.Errorf("%s", strings.Join(msgs, "; "))
}

func friendlyMessage(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, fe.Param())
	case "min":
		return fmt.Sprintf("%s must have at least %s items", field, fe.Param())
	case "valid_status":
		return fmt.Sprintf("%s must be a valid status (approved, rejected, superseded, archived)", field)
	case "valid_algo_key":
		return fmt.Sprintf("%s must be in format name@version", field)
	default:
		return fmt.Sprintf("%s failed validation: %s", field, fe.Tag())
	}
}
