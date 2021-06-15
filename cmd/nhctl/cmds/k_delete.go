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
	"k8s.io/kubectl/pkg/cmd/delete"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os"
)

var deleteFlags = delete.NewDeleteCommandFlags("containing the resource to delete.")

var cmdDelete = &cobra.Command{
	Use:     "delete",
	Example: `nhctl k delete [podName] --namespace nocalhost-reserved --kubeconfig=[path]`,
	Long:    `nhctl k delete [podName] --namespace nocalhost-reserved --kubeconfig=[path]`,
	Short:   `Delete resources by filenames, stdin, resources and names, or by resources and label selector.`,
	Run: func(cmd *cobra.Command, args []string) {
		must(Prepare())
		clientGoUtils, err := clientgoutils.NewClientGoUtils(kubeConfig, nameSpace)
		must(err)
		factory := clientGoUtils.NewFactory()
		deleteOptions, err := deleteFlags.ToOptions(
			clientGoUtils.GetDynamicClient(),
			genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
		)
		cmdutil.CheckErr(err)
		cmdutil.CheckErr(deleteOptions.Complete(factory, args, cmd))
		cmdutil.CheckErr(deleteOptions.Validate())
		cmdutil.CheckErr(deleteOptions.RunDelete(factory))
	},
}

func init() {
	deleteFlags.AddFlags(cmdDelete)
	cmdutil.AddDryRunFlag(cmdDelete)
	kubectlCmd.AddCommand(cmdDelete)
}
