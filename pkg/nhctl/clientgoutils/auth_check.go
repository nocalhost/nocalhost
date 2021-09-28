package clientgoutils

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"nocalhost/internal/nocalhost-api/service/cooperator/util"
	"nocalhost/pkg/nhctl/log"
	"sort"
	"strings"
)

var (
	// holds all the server-supported resources that cannot be discovered by clients. i.e. users and groups for the impersonate verb
	nonStandardResourceNames = sets.NewString("users", "groups")

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
		Verb:        []string{"get", "list", "watch", "create", "patch", "delete"},
		ResourceArg: "Secrets",
	}

	AllPermissionForDeployment = &AuthChecker{
		Verb:        []string{"get", "list", "watch", "create", "patch", "delete"},
		ResourceArg: "Deployments",
	}

	AllPermissionForStatefulSet = &AuthChecker{
		Verb:        []string{"get", "list", "watch", "create", "patch", "delete"},
		ResourceArg: "Statefulsets",
	}

	AllPermissionForDaemonSet = &AuthChecker{
		Verb:        []string{"get", "list", "watch", "create", "patch", "delete"},
		ResourceArg: "Daemonsets",
	}

	AllPermissionForPod = &AuthChecker{
		Verb:        []string{"get", "list", "watch", "create", "patch", "delete"},
		ResourceArg: "pods",
	}

	AllPermissionForJob = &AuthChecker{
		Verb:        []string{"get", "list", "watch", "create", "patch", "delete"},
		ResourceArg: "jobs",
	}

	AllPermissionForCronJob = &AuthChecker{
		Verb:        []string{"get", "list", "watch", "create", "patch", "delete"},
		ResourceArg: "cronjobs",
	}
)

type AuthChecker struct {
	Verb        []string
	Name        string
	Version     string
	ResourceArg string
}

// should call after Prepare()
// fatal if haven't such permission
func DoCheck(cmd *cobra.Command, namespace string, client *ClientGoUtils) error {
	authCheckers := getAuthChecker(cmd)

	mapper, err := client.NewFactory().ToRESTMapper()
	if err != nil {
		return err
	}

	forbiddenCheckers := make([]*authorizationv1.ResourceAttributes, 0)
	if authCheckers != nil {
		for _, checker := range authCheckers {

			for _, verb := range checker.Verb {

				r := resourceFor(mapper, checker.ResourceArg)

				ra := &authorizationv1.ResourceAttributes{
					Namespace: namespace,
					Group:     r.Group,
					Verb:      verb,
					Name:      checker.Name,
					Version:   checker.Version,
					Resource:  r.Resource,
				}

				arg := &authorizationv1.SelfSubjectAccessReview{
					Spec: authorizationv1.SelfSubjectAccessReviewSpec{
						ResourceAttributes: ra,
					},
				}

				resp, err := client.ClientSet.AuthorizationV1().SelfSubjectAccessReviews().
					Create(context.TODO(), arg, metav1.CreateOptions{})

				if err != nil {
					return err
				}

				if !resp.Status.Allowed {
					forbiddenCheckers = append(forbiddenCheckers, ra)
				}
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

func resourceFor(mapper meta.RESTMapper, resourceArg string) schema.GroupVersionResource {
	if resourceArg == "*" {
		return schema.GroupVersionResource{Resource: resourceArg}
	}

	fullySpecifiedGVR, groupResource := schema.ParseResourceArg(strings.ToLower(resourceArg))
	gvr := schema.GroupVersionResource{}
	if fullySpecifiedGVR != nil {
		gvr, _ = mapper.ResourceFor(*fullySpecifiedGVR)
	}
	if gvr.Empty() {
		var err error
		gvr, err = mapper.ResourceFor(groupResource.WithVersion(""))
		if err != nil {
			if !nonStandardResourceNames.Has(groupResource.String()) {
				if len(groupResource.Group) == 0 {
					log.Logf("Warning: the server doesn't have a resource type '%s'\n", groupResource.Resource)
				} else {
					log.Logf(
						"Warning: the server doesn't have a resource type '%s' in group '%s'\n", groupResource.Resource,
						groupResource.Group,
					)
				}
			}
			return schema.GroupVersionResource{Resource: resourceArg}
		}
	}

	return gvr
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
