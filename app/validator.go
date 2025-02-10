package chatter

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
)

var validate *validator.Validate
var uniTrans *ut.UniversalTranslator

func init() {

	validate = validator.New()
	en := en.New()
	uniTrans = ut.New(en, en)
	enTrans, _ := uniTrans.GetTranslator("en")

	validate.RegisterTranslation("hostname", enTrans, func(ut ut.Translator) error {
		return ut.Add("hostname", "{0} must be a valid hostname", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("hostname", fe.Field())
		return t

	})

	// lowercase first letter of the field
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		return strings.ToLower(field.Name)
	})

	validate.RegisterTranslation("required", enTrans, func(ut ut.Translator) error {
		return ut.Add("required", "{0} is a required field", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Field())
		return t
	})

	validate.RegisterTranslation("base64", enTrans, func(ut ut.Translator) error {
		return ut.Add("base64", "{0} must be a valid base64 encoded string", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("base64", fe.Field())
		return t
	})

	validate.RegisterValidation("port", func(fl validator.FieldLevel) bool {
		port, ok := fl.Field().Interface().(int)
		if !ok {
			return false
		}
		return port > 0 && port <= 65535
	})

	validate.RegisterTranslation("port", enTrans, func(ut ut.Translator) error {
		return ut.Add("port", "{0} must be a valid port number", true)

	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("port", fe.Field())
		return t
	})

}
