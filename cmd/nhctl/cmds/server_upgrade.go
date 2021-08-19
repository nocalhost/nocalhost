/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"regexp"
	"strconv"
	"strings"
)

var nhServerVersion string
var wait bool

func init() {
	serverUpgradeCmd.Flags().StringVar(&nhServerVersion, "to-version", "", "The vision to update to")
	serverUpgradeCmd.Flags().BoolVar(&wait, "wait", false, "Waiting deployment to be ready")
	serverCmd.AddCommand(serverUpgradeCmd)
}

var serverUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Nocalhost Server",
	Long:  `Upgrade Nocalhost Server`,
	Run: func(cmd *cobra.Command, args []string) {

		if nhServerVersion == "" {
			log.Fatal("--to-version must be specified")
		}

		pattern := "^v\\d+\\.\\d+\\.\\d+"
		match, _ := regexp.MatchString(pattern, nhServerVersion)
		if !match {
			log.Fatalf("--to-version %s is invalid, no matching: %s", nhServerVersion, `^v\\d+\\.\\d+\\.\\d+`)
		}

		// Find nocalhost server
		client, err := clientgoutils.NewClientGoUtils(kubeConfig,nameSpace)
		must(err)

		// nocalhost-api
		upgradeDeployToVersion(client, "nocalhost-api")

		// nocalhost-web
		upgradeDeployToVersion(client, "nocalhost-web")
	},
}

func upgradeDeployToVersion(client *clientgoutils.ClientGoUtils, deployName string) {
	pattern := "^v\\d+\\.\\d+\\.\\d+"

	deploy, err := client.GetDeployment(deployName)
	if err != nil {
		log.FatalE(err, fmt.Sprintf("Failed to get %s", deployName))
	}

	if len(deploy.Spec.Template.Spec.Containers) == 0 {
		log.Fatalf("Not container found in %s",deployName)
	}

	image := deploy.Spec.Template.Spec.Containers[0].Image
	versionSlice := strings.Split(image,":")
	versionStr := versionSlice[len(versionSlice) -1]

	match, _ := regexp.MatchString(pattern, versionStr)
	if !match {
		log.Fatalf("%s version %s is invalid, no matching: %s", deployName,versionStr, `^v\\d+\\.\\d+\\.\\d+`)
	}

	// Check current version
	if CompareVersion(nhServerVersion, versionStr) <= 0 {
		log.Fatalf("--to-version %s is not larger than %s's version %s", nhServerVersion,deployName ,versionStr)
	}

	// Upgrade
	log.Infof("Upgrading %s to %s", deployName, nhServerVersion)
	versionSlice[len(versionSlice)-1] = nhServerVersion
	newImage := strings.Join(versionSlice,":")
	deploy.Spec.Template.Spec.Containers[0].Image = newImage
	if _, err = client.UpdateDeployment(deploy, wait); err != nil {
		log.FatalE(err, fmt.Sprintf("Failed to update deployment %s", deployName))
	}
}

func CompareVersion(v1,v2 string) int {
	r, _ := regexp.Compile("\\d+")
	v1Nums := r.FindAllString(v1, -1)
	v2Nums := r.FindAllString(v2, -1)
	minLen := len(v1Nums)
	if len(v2Nums) < minLen {
		minLen = len(v2Nums)
	}
	for i:=0; i<minLen; i++{
		v1Int, _ := strconv.ParseInt(v1Nums[i],0, 64)
		v2Int, _ := strconv.ParseInt(v2Nums[i],0, 64)
		if v1Int > v2Int {
			return 1
		}
		if v1Int < v2Int {
			return -1
		}
	}
	if len(v1Nums) > len(v2Nums){
		return 1
	}
	if len(v1Nums) < len(v2Nums){
		return -1
	}
	return 0
}