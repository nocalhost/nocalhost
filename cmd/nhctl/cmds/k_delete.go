/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
