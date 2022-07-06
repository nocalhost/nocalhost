package clientgoutils

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/utils"
	"sort"
	"strings"
)

var (
	// a resource for dev start should check the permission from secret
	// , pod and itself
	//
	// if a resource needs others auth check put it into this mapping
	ExtCheckForResource = map[string][]string{
		"cronjob": {
			"job",
		},
		"daemonset": {
			"deployment",
		},
	}
)

func AuthCheck(namespace, kubeConfig string, cmd *cobra.Command) bool {
	resource := findResourceTypeFromCmd(authCheckCmdRoute(cmd))
	needChecks, _ := ExtCheckForResource[resource]
	if needChecks == nil {
		needChecks = []string{}
	}

	needChecks = append(needChecks, []string{resource, "pod", "secret"}...)

	daemonClient, err := daemon_client.GetDaemonClient(utils.IsSudoUser())
	if err != nil {
		return true
	}

	result, err := daemonClient.SendAuthCheckCommand(namespace, fp.NewFilePath(kubeConfig).ReadFile(), needChecks...)
	return err == nil && result
}

func findResourceTypeFromCmd(cmd []string) string {
	for _, param := range cmd {
		if strings.HasPrefix(param, "controller-type=") {
			results := strings.SplitAfter(param, "controller-type=")
			if len(results) > 1 {
				return results[1]
			}
		}
	}
	return "deployment"
}

func GetCmd(cmd *cobra.Command, from []string) []string {
	var cmdRoute []string
	if from == nil {
		cmdRoute = make([]string, 0)
	} else {
		cmdRoute = from
	}

	parentValid := cmd.HasParent() && cmd.Parent().Name() != ""

	if parentValid {
		cmdRoute = GetCmd(cmd.Parent(), cmdRoute)
	}

	cmdRoute = append(cmdRoute, cmd.Name())
	return cmdRoute
}

func authCheckCmdRoute(cmd *cobra.Command) []string {
	var cmdRoute = GetCmd(cmd, nil)

	// we should get all flags from child cmd
	// filter the flags needed, and append to the end
	// of cmdRoute

	extArgs := make([]string, 0)
	flags := cmd.Flags()
	flags.Visit(
		func(flag *pflag.Flag) {
			extArgs = append(extArgs, fmt.Sprintf("%s=%s", flag.Name, strings.ToLower(flag.Value.String())))
		},
	)

	sort.Slice(
		extArgs, func(i, j int) bool {
			return extArgs[i] > extArgs[j]
		},
	)

	cmdRoute = append(cmdRoute, extArgs...)
	return cmdRoute
}
