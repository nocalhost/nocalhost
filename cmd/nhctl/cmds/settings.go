package cmds

import (
	"os"
	"strconv"
)

type EnvSettings struct {
	Debug bool

	// the path to the kubeconfig file
	KubeConfig string

	Namespace string
}

func NewEnvSettings() *EnvSettings {
	settings := EnvSettings{}
	settings.Debug, _ = strconv.ParseBool(os.Getenv("NOCALHOST_DEBUG"))
	return &settings
}
