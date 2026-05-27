package response

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

func convertToValidationError(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf(prepareFormValidationErrorMessage(err))
}

func prepareFormValidationErrorMessage(err error) string {
	if err.Error() == "EOF" {
		return "no data is provided"
	}

	if err.Error() == "" {
		return "no error content is provided"
	}

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, fe := range validationErrors {
			errorMessage := ""

			switch fe.Tag() {
			case "required":
				errorMessage = fmt.Sprintf("%s is required", fe.Field())
			case "min": //Strings
				errorMessage = fmt.Sprintf("%s should be longer than %s characters", fe.Field(), fe.Param())
			case "max": //Strings
				errorMessage = fmt.Sprintf("%s cannot be longer than %s characters", fe.Field(), fe.Param())
			case "gte": //Int
				errorMessage = fmt.Sprintf("%s should be greater than or equal to %s", fe.Field(), fe.Param())
			case "lte": //Int
				errorMessage = fmt.Sprintf("%s cannot be lower than %s", fe.Field(), fe.Param())
			case "alphanum":
				errorMessage = fmt.Sprintf("%s should be alphanumeric", fe.Field())
			case "oneof":
				errorMessage = fmt.Sprintf("%s is not valid", fe.Field())
			case "email":
				errorMessage = fmt.Sprintf("%s is not a valid email", fe.Field())
			case "len":
				errorMessage = fmt.Sprintf("%s should be %s characters", fe.Field(), fe.Param())
			case "url":
				errorMessage = fmt.Sprintf("%s should be a valid url", fe.Field())
			case "validCountryCode":
				errorMessage = fmt.Sprintf("%s should be a valid country code", fe.Field())
			case "validLanguageCode":
				errorMessage = fmt.Sprintf("%s should be a valid language code", fe.Field())
			case "lowercase":
				errorMessage = fmt.Sprintf("%s should be a lowercase", fe.Field())
			case "gt":
				errorMessage = fmt.Sprintf("%s should be greater than %s", fe.Field(), fe.Param())
			case "notblank":
				errorMessage = fmt.Sprintf("%s can not be blank", fe.Field())
			case "languagecode":
				errorMessage = fmt.Sprintf("%s is not a valid language code", fe.Value())

			default:
				errorMessage = fmt.Sprintf("A problem occured for field %s (%s)", fe.Field(), fe.Tag())
			}

			return errorMessage
		}
	}

	return fmt.Sprintf("undefined validation error: %s", err.Error())
}
