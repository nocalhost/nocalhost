package cmds

import (
	"nocalhost/internal/nhctl/log"
)

//func debug(v ...interface{}) {
//	if settings.Debug {
//		log.Debug(v)
//	}
//}

func debugf(format string, v ...interface{}) {
	if settings.Debug {
		log.Debugf(format, v)
	}
}
