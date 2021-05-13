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
	logs "k8s.io/kubectl/pkg/cmd/logs"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os"
)

var logOptions = logs.NewLogsOptions(
	genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}, false)

var cmdLog = &cobra.Command{
	Use:     "logs",
	Example: `nhctl logs [podName] -c [containerName] -f=true --tail=1 --namespace nocalhost-reserved --kubeconfig=[kubeconfigPath]`,
	Long:    `nhctl logs [podName] -c [containerName] -t [lines] -f true --kubeconfig=[kubeconfigPath]`,
	Short:   ``,
	Run: func(cmd *cobra.Command, args []string) {
		must(Prepare())
		clientGoUtils, err := clientgoutils.NewClientGoUtils(kubeConfig, nameSpace)
		must(err)
		cmdutil.CheckErr(logOptions.Complete(clientGoUtils.NewFactory(), cmd, args))
		cmdutil.CheckErr(logOptions.Validate())
		cmdutil.CheckErr(logOptions.RunLogs())
	}}

func init() {
	logOptions.AddFlags(cmdLog)
	rootCmd.AddCommand(cmdLog)
}
