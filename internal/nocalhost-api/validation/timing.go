package validation

import (
	"github.com/go-playground/validator/v10"
	"regexp"
	"strconv"
	"strings"
)

var Timing validator.Func = func(level validator.FieldLevel) bool {
	text, ok := level.Field().Interface().(string)
	if ok {
		exp, _ := regexp.Compile(`^(\d{2}):(\d{2})$`)
		yes := exp.MatchString(text)
		if yes {
			h, err := strconv.Atoi(strings.Split(text, ":")[0])
			if err != nil {
				return false
			}
			m, err := strconv.Atoi(strings.Split(text, ":")[1])
			if err != nil {
				return false
			}
			if h >= 0 && h <= 23 && m >= 0 && m <= 59 {
				return true
			}
		}
	}
	return false
}
