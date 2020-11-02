package cmds

import "fmt"

func debug(format string, v ...interface{}) {
	if settings.Debug {
		format = fmt.Sprintf("[debug] %s\n", format)
		fmt.Printf(format, v...)
	}
}
