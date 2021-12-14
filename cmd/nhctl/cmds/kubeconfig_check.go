/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
)

var (
	contextSpecified []string
	interactive      bool
)

func init() {
	kubeconfigCmd.AddCommand(kubeconfigCheckCmd)
	kubeconfigCheckCmd.Flags().StringArrayVarP(&contextSpecified,
		"context", "c", []string{},
		"By default, the current context is used. If there is no cluster scope permission, check the 'namespace' is specified",
	)
	kubeconfigCheckCmd.Flags().BoolVarP(&interactive,
		"interactive", "i", false, "return readable interactive result")
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

		checkInfos := make([]CheckInfo, 0)

		notEmpty := false
		warnMsg := "Your KubeConfig may illegal, please try to fix it by following the tips below: <br>"
		for _, context := range contextSpecified {
			checkInfo := CheckKubeconfig(kubeConfig, context)
			checkInfos = append(checkInfos, checkInfo)

			if checkInfo.Hint != "" {
				notEmpty = true
				warnMsg += fmt.Sprintf("%s <br>", checkInfo.Hint)
			}
		}

		if interactive {
			marshal, _ := json.Marshal(checkInfos)
			fmt.Println(string(marshal))
		} else {
			if notEmpty {
				log.PWarn(warnMsg)
			}
		}
	},
}

// # CheckKubeconfig return two strings
// first is the complate guide
// second is a simple msg
func CheckKubeconfig(kubeconfigParams string, contextParam string) CheckInfo {
	utils, err := clientgoutils.NewClientGoUtils(kubeconfigParams, "")
	if err != nil {
		return CheckInfo{
			FAIL, InfoInvalid, err.Error(), true, err.Error(),
		}
	}

	config, err := utils.NewFactory().ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return CheckInfo{
			FAIL, InfoInvalid, err.Error(), true, err.Error(),
		}
	}

	if len(config.Contexts) == 0 {
		msg := "Please make sure your kubeconfig contains one context at least "
		return CheckInfo{
			FAIL, InfoInvalid, msg, true, msg,
		}
	}

	if contextParam == "" {
		contextParam = config.CurrentContext
	}

	if ctx, ok := config.Contexts[contextParam]; !ok {
		set := sets.NewString()
		for validContext, _ := range config.Contexts {
			set.Insert(validContext)
		}

		msg := fmt.Sprintf(
			"Invalid context '%s',"+
				" you should choose a correct one from below %v",
			contextParam, set.UnsortedList(),
		)
		return CheckInfo{
			FAIL, InfoInvalid, msg, true, msg,
		}
	} else {
		specifiedNs := ctx.Namespace == ""

		err := clientgoutils.DoAuthCheck(
			utils, "", &clientgoutils.AuthChecker{
				Verb:        []string{"list", "get", "watch"},
				ResourceArg: "namespaces",
			},
		)

		if err != nil {
			if errors.Is(err, clientgoutils.PermissionDenied) {
				msg := fmt.Sprintf(
					"Context '%s' can not asscess the cluster scope resources, so you should specify a namespace by using "+
						"'kubectl config set-context %s --namespace=${your_namespace} --kubeconfig=${your_kubeconfig}', or you can add"+
						" a namespace to this context manually. ",
					contextParam, contextParam,
				)

				if specifiedNs{
					return CheckInfo{
						SUCCESS, ctx.Namespace, fmt.Sprintf("Current context can list the resources behind ns %s", ctx.Namespace),
						true, "",
					}
				}

				return CheckInfo{
					FAIL, "Type in a namespace", "Please Type in a correct namespace and try again", true, msg,
				}
			}
			return CheckInfo{
				FAIL, "Type in a namespace", "Please Type in a correct namespace and try again", true, err.Error(),
			}
		}

		// Success with cluster scope
		return CheckInfo{
			SUCCESS, "Cluster-Scope", "Current context can list with cluster-scope resources", false, "",
		}
	}
}

var (
	SUCCESS CheckInfoStatus = "SUCCESS"
	FAIL    CheckInfoStatus = "FAIL"

	InfoInvalid = "invalid"
)

type CheckInfo struct {
	Status CheckInfoStatus `json:"status" yaml:"status"`
	Info   string          `json:"info" yaml:"info"`
	Tips   string          `json:"tips" yaml:"tips"`
	TypeIn bool            `json:"typein" yaml:"typein"`
	Hint   string          `json:"hint" yaml:"hint"`
}

type CheckInfoStatus string
