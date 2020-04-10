package common

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/gobeam/golang-oauth/example/core/models"
	"github.com/jinzhu/gorm"
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
		v.RegisterValidation("uniqueEmail", uniqueEmail)
	}
}

var uniqueEmail validator.Func = func(fl validator.FieldLevel) bool {
	user := models.User{}
	models.DB.Where(&models.User{
		Email: fl.Field().String(),
	}).First(&user)
	if user.ID == 0 {
		return true
	}
	return false
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
