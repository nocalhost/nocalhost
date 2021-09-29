/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	dockerterm "github.com/moby/term"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/exec"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/pkg/nhctl/clientgoutils"
	"time"
)

var options = &exec.ExecOptions{
	Executor: &exec.DefaultRemoteExecutor{},
}

var cmdExec = &cobra.Command{
	Use:     "exec",
	Example: "exec (POD | TYPE/NAME) [-c CONTAINER] [flags] -- COMMAND [args...]",
	Long:    `Execute a command in a container`,
	Short:   `Execute a command in a container`,
	Run: func(cmd *cobra.Command, args []string) {
		clientGoUtils, err := clientgoutils.NewClientGoUtils(kubeConfig, nameSpace)
		must(err)
		stdin, stdout, stderr := dockerterm.StdStreams()
		options.IOStreams = genericclioptions.IOStreams{In: stdin, Out: stdout, ErrOut: stderr}
		cmdutil.AddPodRunningTimeoutFlag(cmd, 60*time.Second)
		cmdutil.AddJsonFilenameFlag(cmd.Flags(), &options.FilenameOptions.Filenames, "to use to exec into the resource")
		argsLenAtDash := cmd.ArgsLenAtDash()
		cmdutil.CheckErr(options.Complete(clientGoUtils.NewFactory(), cmd, args, argsLenAtDash))
		cmdutil.CheckErr(options.Validate())
		cmdutil.CheckErr(options.Run())
	},
}

func init() {
	cmdExec.Flags().StringVarP(&options.ContainerName, "container", "c", options.ContainerName, "Container name. If omitted, the first container in the pod will be chosen")
	cmdExec.Flags().BoolVarP(&options.Stdin, "stdin", "i", options.Stdin, "Pass stdin to the container")
	cmdExec.Flags().BoolVarP(&options.TTY, "tty", "t", options.TTY, "Stdin is a TTY")
	kubectlCmd.AddCommand(cmdExec)
}
