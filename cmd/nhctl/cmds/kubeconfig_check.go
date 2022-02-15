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
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/sets"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/k8sutils"
	"nocalhost/pkg/nhctl/log"
	"os"
)

var (
	contextSpecified []string
	interactive      bool
)

func init() {
	kubeconfigCmd.AddCommand(kubeconfigCheckCmd)
	kubeconfigCheckCmd.Flags().StringArrayVarP(
		&contextSpecified,
		"context", "c", []string{},
		"By default, the current context is used. If there is no cluster scope permission, check the 'namespace' is specified",
	)
	kubeconfigCheckCmd.Flags().BoolVarP(
		&interactive,
		"interactive", "i", false, "return readable interactive result",
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

		if common.KubeConfig == "-" { // from sdtin

			// TODO: Consider adding a flag to force to UTF16, apparently some
			// Windows tools don't write the BOM
			utf16bom := unicode.BOMOverride(unicode.UTF8.NewDecoder())
			reader := transform.NewReader(os.Stdin, utf16bom)

			content, err := ioutil.ReadAll(reader)
			must(err)

			common.KubeConfig = k8sutils.GetOrGenKubeConfigPath(string(content))

			defer fp.NewFilePath(common.KubeConfig).Remove()
		}

		checkInfos := make([]CheckInfo, 0)

		notEmpty := false
		warnMsg := "Your KubeConfig may illegal, please try to fix it by following the tips below: <br>"
		for _, context := range contextSpecified {
			checkInfo := CheckKubeconfig(common.KubeConfig, context)
			checkInfos = append(checkInfos, checkInfo)

			if checkInfo.Status == FAIL {
				notEmpty = true
				warnMsg += fmt.Sprintf("%s <br>", checkInfo.Tips)
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
	unexpectedErrFmt := "Please check if your cluster or kubeconfig is valid, Error: %v"
	invalidNamespaceFmt := "Please check current namespace is valid, or make sure" +
		" you have the correct permissions to access this namespace, Error: %v"

	utils, err := clientgoutils.NewClientGoUtils(kubeconfigParams, "")
	if err != nil {
		return CheckInfo{
			FAIL, fmt.Sprintf(unexpectedErrFmt, err.Error()),
		}
	}

	config, err := utils.NewFactory().ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return CheckInfo{
			FAIL, fmt.Sprintf(unexpectedErrFmt, err.Error()),
		}
	}

	if len(config.Contexts) == 0 {
		return CheckInfo{FAIL, "Please make sure your kubeconfig contains at least one context"}
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
			FAIL, msg,
		}
	} else {
		specifiedNs := ctx.Namespace != ""
		_ = common.Prepare()

		kubeCOnfigContent := fp.NewFilePath(common.KubeConfig).ReadFile()

		checkChanForClusterScope := make(chan error, 0)
		checkChanForNsScope := make(chan error, 0)

		go func() {
			checkChanForClusterScope <- clientgoutils.CheckForResource(
				kubeCOnfigContent, "", []string{"list", "get", "watch"}, false, "namespaces",
			)
		}()

		go func() {
			checkChanForNsScope <- clientgoutils.CheckForResource(
				kubeCOnfigContent, ctx.Namespace, []string{"list"}, false, "pod",
			)
		}()

		select {
		case errNs := <-checkChanForNsScope:
			{
				if errNs != nil {
					return CheckInfo{FAIL, fmt.Sprintf(invalidNamespaceFmt, errNs.Error())}
				}
				return CheckInfo{SUCCESS, ""}
			}

		case errCl := <-checkChanForClusterScope:
			{
				if errCl != nil {
					if errors.Is(errCl, clientgoutils.PermissionDenied) {
						msg := fmt.Sprintf(
							"Context '%s' can not asscess the cluster scope resources, so you should specify a namespace by using "+
								"'kubectl config set-context %s --namespace=${your_namespace} --kubeconfig=${your_kubeconfig}', or you can add"+
								" a namespace to this context manually. ",
							contextParam, contextParam,
						)

						if specifiedNs {
							if err := <-checkChanForNsScope; err != nil {
								return CheckInfo{FAIL, fmt.Sprintf(invalidNamespaceFmt, err.Error())}
							}
							return CheckInfo{SUCCESS, ""}
						}
						return CheckInfo{FAIL, msg}
					}
					return CheckInfo{FAIL, fmt.Sprintf(unexpectedErrFmt, errCl.Error())}
				}

				// Success with cluster scope
				return CheckInfo{SUCCESS, "Current context can list with cluster-scope resources"}
			}
		}
	}
}


var (
	SUCCESS CheckInfoStatus = "SUCCESS"
	FAIL    CheckInfoStatus = "FAIL"
)

type CheckInfo struct {
	Status CheckInfoStatus `json:"status" yaml:"status"`
	Tips   string          `json:"tips" yaml:"tips"`
}

type CheckInfoStatus string
