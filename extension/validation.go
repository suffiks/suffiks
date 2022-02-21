package extension

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/suffiks/suffiks/extension/protogen"
)

type ValidatorResponse struct {
	validate *validator.Validate
}

func (v *ValidatorResponse) AutoValidate(obj any) (*protogen.ValidationResponse, error) {
	err := v.validate.Struct(obj)
	if err == nil {
		return nil, nil
	}

	if _, ok := err.(*validator.InvalidValidationError); ok {
		return nil, err
	}

	vr := &protogen.ValidationResponse{}
	for _, err := range err.(validator.ValidationErrors) {
		vr.Errors = append(vr.Errors, &protogen.ValidationError{
			Path:   err.Field(),
			Value:  fmt.Sprint(err.Value()),
			Detail: err.Error(),
		})
	}

	return vr, nil
}
