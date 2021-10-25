/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
)

var (
	contextSpecified []string
)

func init() {
	kubeconfigCmd.AddCommand(kubeconfigCheckCmd)
	kubeconfigCheckCmd.Flags().StringArrayVarP(&contextSpecified,
		"context", "c", []string{},
		"By default, the current context is used. If there is no cluster scope permission, check the 'namespace' is specified",
	)
}

// Check kubeconfig file, if the kubeconfig does have a 'ns list' permission, return nothing
// or else return error code 1
// if error occur, not only return error code
// and show the reason to stderr
var kubeconfigCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "check kubeconfig",
	Long:  `check kubeconfig`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(contextSpecified) == 0 {
			contextSpecified = append(contextSpecified, "")
		}

		notEmpty := false
		warnMsg := "Your KubeConfig may illegal, please try to fix it by following the tips below: <br>"
		for _, context := range contextSpecified {

			if tips := CheckKubeconfig(kubeConfig, context); tips != "" {
				notEmpty = true
				warnMsg += fmt.Sprintf("%s <br>", tips)
			}
		}

		if notEmpty {
			log.PWarn(warnMsg)
		}
	},
}

func CheckKubeconfig(kubeconfigParams string, contextParam string) string {
	utils, err := clientgoutils.NewClientGoUtils(kubeconfigParams, "")
	if err != nil {
		return err.Error()

	}

	config, err := utils.NewFactory().ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err.Error()

	}

	if len(config.Contexts) == 0 {
		return "Please make sure your kubeconfig contains one context at least "
	}

	if contextParam == "" {
		contextParam = config.CurrentContext
	}

	if ctx, ok := config.Contexts[contextParam]; !ok {
		set := sets.NewString()
		for validContext, _ := range config.Contexts {
			set.Insert(validContext)
		}

		return fmt.Sprintf(
			"Invalid context '%s',"+
				" you should choose a correct one from below %v",
			contextParam, set.UnsortedList(),
		)

	} else {
		if ctx.Namespace == "" {

			err := clientgoutils.DoAuthCheck(
				utils, "", &clientgoutils.AuthChecker{
					Verb:        []string{"list", "get", "watch"},
					ResourceArg: "namespaces",
				},
			)

			if errors.Is(err, clientgoutils.PermissionDenied) {

				return fmt.Sprintf(
					"Context '%s' can not asscess the cluster scope resources, so you should specify a namespace by using "+
						"'kubectl config set-context %s --namespace=${your_namespace} --kubeconfig=${your_kubeconfig}', or you can add"+
						" a namespace to this context manually. ",
					contextParam, contextParam,
				)
			}
		}
	}

	return ""
}
