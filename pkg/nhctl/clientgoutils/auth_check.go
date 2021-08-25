package clientgoutils

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nocalhost-api/service/cooperator/util"
	"nocalhost/pkg/nhctl/log"
	"sort"
	"strings"
)

var (
	CheckersMapping = map[string][]*AuthChecker{
		"nhctl dev start controller-type=deployment": {
			AllPermissionForSecret,
			AllPermissionForDeployment,
			AllPermissionForPod,
		},
		"nhctl dev start controller-type=statefulset": {
			AllPermissionForSecret,
			AllPermissionForStatefulSet,
			AllPermissionForPod,
		},
		"nhctl dev start controller-type=daemonset": {
			AllPermissionForSecret,
			AllPermissionForDeployment,
			AllPermissionForDaemonSet,
			AllPermissionForPod,
		},
		"nhctl dev start controller-type=job": {
			AllPermissionForSecret,
			AllPermissionForJob,
			AllPermissionForPod,
		},
		"nhctl dev start controller-type=cronjob": {
			AllPermissionForSecret,
			AllPermissionForCronJob,
			AllPermissionForJob,
			AllPermissionForPod,
		},
		"nhctl dev start controller-type=pod": {
			AllPermissionForSecret,
			AllPermissionForPod,
		},

		// install may use different type's create permissions
		// but we can not know in advance, so there could only
		// check secret's permissions
		"nhctl install": {
			AllPermissionForSecret,
		},
	}

	ArgsFilter = util.NewSet("controller-type")

	AllPermissionForSecret = &AuthChecker{
		Group:    "*",
		Verb:     "*",
		Name:     "*",
		Version:  "*",
		Resource: "secrets",
	}

	AllPermissionForDeployment = &AuthChecker{
		Group:    "*",
		Verb:     "*",
		Name:     "*",
		Version:  "*",
		Resource: "deployments",
	}

	AllPermissionForStatefulSet = &AuthChecker{
		Group:    "*",
		Verb:     "*",
		Name:     "*",
		Version:  "*",
		Resource: "statefulsets",
	}

	AllPermissionForDaemonSet = &AuthChecker{
		Group:    "*",
		Verb:     "*",
		Name:     "*",
		Version:  "*",
		Resource: "daemonsets",
	}

	AllPermissionForPod = &AuthChecker{
		Group:    "*",
		Verb:     "*",
		Name:     "*",
		Version:  "*",
		Resource: "pods",
	}

	AllPermissionForJob = &AuthChecker{
		Group:    "*",
		Verb:     "*",
		Name:     "*",
		Version:  "*",
		Resource: "jobs",
	}

	AllPermissionForCronJob = &AuthChecker{
		Group:    "*",
		Verb:     "*",
		Name:     "*",
		Version:  "*",
		Resource: "cronjobs",
	}
)

type AuthChecker struct {
	Group    string
	Verb     string
	Name     string
	Version  string
	Resource string
}

// should call after Prepare()
// fatal if haven't such permission
func DoCheck(cmd *cobra.Command, namespace string, client *ClientGoUtils) error {
	authCheckers := getAuthChecker(cmd)

	forbiddenCheckers := make([]*AuthChecker, 0)
	if authCheckers != nil {
		for _, checker := range authCheckers {
			arg := &authorizationv1.SelfSubjectAccessReview{
				Spec: authorizationv1.SelfSubjectAccessReviewSpec{
					ResourceAttributes: &authorizationv1.ResourceAttributes{
						Namespace: namespace,
						Group:     checker.Group,
						Verb:      checker.Verb,
						Name:      checker.Name,
						Version:   checker.Version,
						Resource:  checker.Resource,
					},
				},
			}

			resp, err := client.ClientSet.AuthorizationV1().SelfSubjectAccessReviews().
				Create(context.TODO(), arg, metav1.CreateOptions{})

			if err != nil {
				return err
			}

			if !resp.Status.Allowed {
				forbiddenCheckers = append(forbiddenCheckers, checker)
			}
		}
	}

	if len(forbiddenCheckers) > 0 {
		marshal, _ := yaml.Marshal(forbiddenCheckers)
		log.Fatal(
			fmt.Sprintf(
				"Permission denied when pre check! "+
					"please make sure you have such permission in current namespace: \n\n%s", marshal,
			),
		)
	}

	return nil
}

func getAuthChecker(cmd *cobra.Command) []*AuthChecker {
	route := authCheckCmdRoute(cmd)

	checkersKey := strings.Join(route, " ")
	checkers, ok := CheckersMapping[checkersKey]

	if !ok {
		log.Fatal("The current command does not implement permission check")
	}
	return checkers
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
			if ArgsFilter.Exist(flag.Name) {
				extArgs = append(extArgs, fmt.Sprintf("%s=%s", flag.Name, strings.ToLower(flag.Value.String())))
			}
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
