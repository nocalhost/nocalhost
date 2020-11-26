package log

import (
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func init() {
	log.SetFormatter(&nested.Formatter{
		HideKeys: true,
		//FieldsOrder: []string{"component", "category"},
	})
}

func SetLevel(level logrus.Level) {
	log.SetLevel(level)
}

func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args)
}

func Fatal(args ...interface{}) {
	log.Fatal(args)
}

func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args)
}

func Debug(args ...interface{}) {
	log.Debug(args)
}
