/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

func Warn(args ...interface{}) {
	log.Warn(args)
}

func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args)
}

func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args)
}

func Info(args ...interface{}) {
	log.Info(args)
}

func Debug(args ...interface{}) {
	log.Debug(args)
}
