/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	rootCmd.AddCommand(cleanupCmd)
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "upgrade k8s application",
	Long:  `upgrade k8s application`,
	Run: func(cmd *cobra.Command, args []string) {
		appMap, err := nocalhost.GetNsAndApplicationInfo()
		if err != nil {
			log.FatalE(err, "")
		}

		// remove useless profile
		for _, a := range appMap {
			log.Infof("Processing app %s, nid %s, namespace %s", a.Name, a.Nid, a.Namespace)
			kube, err := nocalhost.GetKubeConfigFromProfile(a.Namespace, a.Name, a.Nid)
			if err != nil {
				log.WarnE(err, fmt.Sprintf("Failed to get kube for app %s, nid %s, namespace %s", a.Name, a.Nid, a.Namespace))
				continue
			}
			nocalhostApp, err := app.NewApplication(a.Name, a.Namespace, kube, true)
			if err != nil {
				log.WarnE(err, fmt.Sprintf("Failed to newApplication for app %s, nid %s, namespace %s", a.Name, a.Nid, a.Namespace))
				continue
			}
			if nocalhostApp.GetAppMeta().NamespaceId != a.Nid {
				log.Infof("Remove application %s, nid %s, ns %s", a.Name, a.Nid, a.Namespace)
				if err = nocalhost.CleanupAppFilesUnderNs(a.Namespace, a.Nid); err != nil {
					log.Infof("Clean application %s, nid %s, ns %s failed: %s ", a.Name, a.Nid, a.Namespace, err.Error())
				} else {
					log.Infof("Clean application %s, nid %s, ns %s success", a.Name, a.Nid, a.Namespace)
				}
			} else {
				log.Infof("No need to processing app %s, nid %s, namespace %s", a.Name, a.Nid, a.Namespace)
			}
		}
	},
}
