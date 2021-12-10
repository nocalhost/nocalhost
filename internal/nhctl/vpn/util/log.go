package util

import (
	"fmt"
	log "github.com/sirupsen/logrus"
)

func InitLogger(debug bool) {
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	log.SetReportCaller(true)
	log.SetFormatter(&Format{})
}

type Format struct {
	log.Formatter
}

//	2009/01/23 01:23:23 d.go:23: message
// same like log.SetFlags(log.LstdFlags | log.Lshortfile)
func (*Format) Format(e *log.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf("%s\n", e.Message)), nil
}
