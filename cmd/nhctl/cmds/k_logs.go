/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	dockerterm "github.com/moby/term"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/logs"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/pkg/nhctl/clientgoutils"
)

var logOptions *logs.LogsOptions

var cmdLogs = &cobra.Command{
	Use:     "logs",
	Example: `nhctl logs [podName] -c [containerName] -f=true --tail=1 --namespace nocalhost-reserved --kubeconfig=[path]`,
	Long:    `nhctl logs [podName] -c [containerName] -t [lines] -f true --kubeconfig=[kubeconfigPath]`,
	Short:   `Print the logs for a container in a pod or specified resource`,
	Run: func(cmd *cobra.Command, args []string) {
		must(common.Prepare())
		clientGoUtils, err := clientgoutils.NewClientGoUtils(common.KubeConfig, common.NameSpace)
		must(err)
		cmdutil.CheckErr(logOptions.Complete(clientGoUtils.NewFactory(), cmd, args))
		cmdutil.CheckErr(logOptions.Validate())
		cmdutil.CheckErr(logOptions.RunLogs())
	}}

func init() {
	stdIn, stdOut, stderr := dockerterm.StdStreams()
	logOptions = logs.NewLogsOptions(
		genericclioptions.IOStreams{In: stdIn, Out: stdOut, ErrOut: stderr}, false)
	logOptions.AddFlags(cmdLogs)
	kubectlCmd.AddCommand(cmdLogs)
}
