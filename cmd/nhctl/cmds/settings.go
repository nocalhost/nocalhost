package cmds

import (
	"os"
	"strconv"
)

type EnvSettings struct {
	Debug bool
}


func NewEnvSettings() *EnvSettings {
	settings := EnvSettings{
	}
	settings.Debug, _ = strconv.ParseBool(os.Getenv("NOCALHOST_DEBUG"))
	return &settings
}