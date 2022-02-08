/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package kube

import (
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/cmd/logs"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/pkg/nhctl/clientgoutils"
)

var LogOptions *logs.LogsOptions

var CmdLogs = &cobra.Command{
	Use:     "logs",
	Example: `nhctl logs [podName] -c [containerName] -f=true --tail=1 --namespace nocalhost-reserved --kubeconfig=[path]`,
	Long:    `nhctl logs [podName] -c [containerName] -t [lines] -f true --kubeconfig=[kubeconfigPath]`,
	Short:   `Print the logs for a container in a pod or specified resource`,
	Run: func(cmd *cobra.Command, args []string) {
		RunLogs(cmd, args)
	}}

func init() {
	//stdIn, stdOut, stderr := dockerterm.StdStreams()
	//LogOptions = logs.NewLogsOptions(
	//	genericclioptions.IOStreams{In: stdIn, Out: stdOut, ErrOut: stderr}, false)
	//LogOptions.AddFlags(CmdLogs)
	InitLogOptions()
}

var flagAdded bool

func InitLogOptions() {
	//stdIn, stdOut, stderr := dockerterm.StdStreams()
	LogOptions = logs.NewLogsOptions(
		*clientgoutils.IoStreams, false)
	if !flagAdded {
		flagAdded = true
		LogOptions.AddFlags(CmdLogs)
	}
}

func RunLogs(cmd *cobra.Command, args []string) {
	common.Must(common.Prepare())
	clientGoUtils, err := clientgoutils.NewClientGoUtils(common.KubeConfig, common.NameSpace)
	common.Must(err)
	cmdutil.CheckErr(LogOptions.Complete(clientGoUtils.NewFactory(), cmd, args))
	cmdutil.CheckErr(LogOptions.Validate())
	cmdutil.CheckErr(LogOptions.RunLogs())
}
