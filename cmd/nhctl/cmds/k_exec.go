/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/exec"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os"
	"time"
)

var options = &exec.ExecOptions{
	StreamOptions: exec.StreamOptions{
		IOStreams: genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
	},
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
