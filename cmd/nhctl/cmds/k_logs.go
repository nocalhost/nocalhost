/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/logs"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os"
)

var logOptions = logs.NewLogsOptions(
	genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}, false)

var cmdLogs = &cobra.Command{
	Use:     "logs",
	Example: `nhctl logs [podName] -c [containerName] -f=true --tail=1 --namespace nocalhost-reserved --kubeconfig=[path]`,
	Long:    `nhctl logs [podName] -c [containerName] -t [lines] -f true --kubeconfig=[kubeconfigPath]`,
	Short:   `Print the logs for a container in a pod or specified resource`,
	Run: func(cmd *cobra.Command, args []string) {
		must(Prepare())
		clientGoUtils, err := clientgoutils.NewClientGoUtils(kubeConfig, nameSpace)
		must(err)
		cmdutil.CheckErr(logOptions.Complete(clientGoUtils.NewFactory(), cmd, args))
		cmdutil.CheckErr(logOptions.Validate())
		cmdutil.CheckErr(logOptions.RunLogs())
	}}

func init() {
	logOptions.AddFlags(cmdLogs)
	kubectlCmd.AddCommand(cmdLogs)
}
