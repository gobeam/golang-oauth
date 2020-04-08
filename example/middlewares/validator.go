package middleware

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stvp/rollbar"
	"gopkg.in/go-playground/validator.v8"
	"net/http"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrorInternalError = errors.New("whoops something went wrong")
)

func UcFirst(str string) string {
	for i, v := range str {
		return string(unicode.ToUpper(v)) + str[i+1:]
	}
	return ""
}

func LcFirst(str string) string {
	return strings.ToLower(str)
}

func Split(src string) string {
	// don't split invalid utf8
	if !utf8.ValidString(src) {
		return src
	}
	var entries []string
	var runes [][]rune
	lastClass := 0
	class := 0
	// split into fields based on class of unicode character
	for _, r := range src {
		switch true {
		case unicode.IsLower(r):
			class = 1
		case unicode.IsUpper(r):
			class = 2
		case unicode.IsDigit(r):
			class = 3
		default:
			class = 4
		}
		if class == lastClass {
			runes[len(runes)-1] = append(runes[len(runes)-1], r)
		} else {
			runes = append(runes, []rune{r})
		}
		lastClass = class
	}

	for i := 0; i < len(runes)-1; i++ {
		if unicode.IsUpper(runes[i][0]) && unicode.IsLower(runes[i+1][0]) {
			runes[i+1] = append([]rune{runes[i][len(runes[i])-1]}, runes[i+1]...)
			runes[i] = runes[i][:len(runes[i])-1]
		}
	}
	// construct []string from results
	for _, s := range runes {
		if len(s) > 0 {
			entries = append(entries, string(s))
		}
	}

	for index, word := range entries {
		if index == 0 {
			entries[index] = UcFirst(word)
		} else {
			entries[index] = LcFirst(word)
		}
	}
	justString := strings.Join(entries, " ")
	return justString
}

func ValidationErrorToText(e *validator.FieldError) string {
	word := Split(e.Field)

	switch e.Tag {
	case "required":
		return fmt.Sprintf("%s is required", word)
	case "max":
		return fmt.Sprintf("%s cannot be longer than %s characters", word, e.Param)
	case "min":
		return fmt.Sprintf("%s must be minimum %s characters", word, e.Param)
	case "email":
		return fmt.Sprintf("Invalid email format")
	case "len":
		return fmt.Sprintf("%s must be %s characters long", word, e.Param)
	case "uniqueEmail":
		return fmt.Sprintf("%s already taken", word)
	case "eqfield":
		return fmt.Sprintf("%s does not match", word)
	case "unique":
		return fmt.Sprintf("%s already taken", word)
	case "password":
		return fmt.Sprintf("%s must be at least one number, one upper case and one special character", word)
	case "nonumstart":
		return fmt.Sprintf("%s cannot start with number", word)
	case "numeric":
		return fmt.Sprintf("%s should only contain numeric characters", word)
	case "phone":
		return fmt.Sprintf("%s should only contain numeric characters", word)
	case "eth_address":
		return fmt.Sprintf("%s must be valid ethereum address", word)
	case "string":
		return fmt.Sprintf("%s must be string", word)
	}
	return fmt.Sprintf("%s is not valid", word)
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// This method collects all errors and submits them to Rollbar
func Errors() gin.HandlerFunc {

	return func(c *gin.Context) {
		// Only run if there are some errors to handle
		c.Next()
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				// Find out what type of error it is
				switch e.Type {
				case gin.ErrorTypePublic:
					// Only output public errors if nothing has been written yet
					//if !c.Writer.Written() {
					//	c.JSON(c.Writer.Status(), gin.H{"Error": e.Error()})
					//}
				case gin.ErrorTypeBind:

					errs := e.Err.(validator.ValidationErrors)
					list := make(map[string]string)

					for _, err := range errs {
						list[strings.ToLower(ToSnakeCase(err.Field))] = ValidationErrorToText(err)
					}

					// Make sure we maintain the preset response status
					status := http.StatusUnprocessableEntity
					//if c.Writer.Status() != http.StatusOK {
					//	status = c.Writer.Status()
					//}
					c.JSON(status, gin.H{"error": list})
					c.Abort()
					return
				default:
					// Log all other errors
					rollbar.RequestError(rollbar.ERR, c.Request, e.Err)
				}

			}
			// If there was no public or bind error, display default 500 message
			if !c.Writer.Written() {
				c.JSON(http.StatusInternalServerError, gin.H{"error": ErrorInternalError.Error()})
			}
		}
	}
}
