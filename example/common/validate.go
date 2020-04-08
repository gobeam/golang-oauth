package common

import (
	"fmt"
	"github.com/gin-gonic/gin/binding"
	"github.com/jinzhu/gorm"
	"github.com/gobeam/golang-oauth/example/core/models"
	"gopkg.in/go-playground/validator.v8"
	"reflect"
)

type ValidatorRegister struct {
	db *gorm.DB
}

func NewValidatorRegister(db *gorm.DB) ValidatorRegister {
	return ValidatorRegister{db}
}

func (valReg ValidatorRegister) RegisterValidator() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		err := v.RegisterValidation("uniqueEmail", valReg.uniqueEmail)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

func (valReg ValidatorRegister) uniqueEmail(
	v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value,
	field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string,
) bool {

	var user models.User
	valReg.db.Where(&models.User{
		Email: field.String(),
	}).First(&user)

	if user.ID == 0 {
		return true
	}
	return false
}
