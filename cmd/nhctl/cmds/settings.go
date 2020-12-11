package cmds

import (
	"os"
	"strconv"
)

type EnvSettings struct {
	Debug      bool
	KubeConfig string // the path to the kubeconfig file
	Namespace  string
}

func NewEnvSettings() *EnvSettings {
	settings := EnvSettings{}
	settings.Debug, _ = strconv.ParseBool(os.Getenv("NOCALHOST_DEBUG"))
	return &settings
}
