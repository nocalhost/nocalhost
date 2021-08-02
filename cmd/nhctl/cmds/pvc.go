/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(pvcCmd)
}

// Managing PersistVolumeClaims of a application, Please make sure the application is installed
// If application is not installed, nhctl is unable to manage PersistVolumeClaims created by the application,
// since it has no the kubeconfig to the k8s cluster
var pvcCmd = &cobra.Command{
	Use:   "pvc",
	Short: "Manage PersistVolumeClaims",
	Long:  `Manage PersistVolumeClaims`,
}
