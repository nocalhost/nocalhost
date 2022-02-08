/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"encoding/json"
	"fmt"
	yaml2 "github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"path/filepath"
	"strings"
)

func init() {
	kubeconfigCmd.AddCommand(kubeconfigCreateCmd)
}

var kubeconfigCreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"c"},
	Short:   "Create a kubeconfig for specified namespace",
	Long:    `Create a kubeconfig for specified namespace`,
	Run: func(cmd *cobra.Command, args []string) {
		if common.KubeConfig == "" {
			common.KubeConfig = filepath.Join(utils.GetHomePath(), ".kube", "config")
		}
		GenKubeconfig(common.KubeConfig, common.NameSpace)
	},
}

func GenKubeconfig(kube, ns string) {
	if ns == "" {
		uid, err := uuid.NewUUID()
		must(err)
		id := strings.Split(uid.String(), "-")[0]
		ns = fmt.Sprintf("nh%s", id)
	}
	k8sClient, err := clientgoutils.NewClientGoUtils(kube, ns)
	must(err)

	labels := map[string]string{"nocalhost.dev/generated-by": "nocalhost"}
	k8sClient = k8sClient.Labels(labels)

	must(k8sClient.CreateNamespaceINE(ns))

	uid, err := uuid.NewUUID()
	must(err)
	id := strings.Split(uid.String(), "-")[0]

	saName := fmt.Sprintf("nocalhost-generated-account-%s", id)
	rb := "nocalhost-generated-role-binding"
	role := "nocalhost-generated-role"

	_, err = k8sClient.CreateAdminClusterRoleINE(role)
	must(err)

	_, err = k8sClient.CreateServiceAccountINE(saName)
	must(err)

	_, err = k8sClient.CreateRoleBindingWithClusterRoleINE(rb, role)
	must(err)

	must(k8sClient.AddClusterRoleToRoleBinding(rb, role, saName))

	restConfig, err := clientcmd.BuildConfigFromFlags("", kube)
	must(err)
	serverAddr := restConfig.Host

	sa, err := k8sClient.GetServiceAccount(saName)
	if err != nil || len(sa.Secrets) == 0 {
		return
	}

	secret, err := k8sClient.GetSecret(sa.Secrets[0].Name)
	must(err)

	ca := secret.Data["ca.crt"]
	if len(ca) == 0 {
		log.Fatal("Failed to get ca")
	}

	token := string(secret.Data["token"])
	if token == "" {
		log.Fatal("Failed to get token")
	}

	kubeStruct := &clientcmdapi.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []clientcmdapi.NamedCluster{
			{
				Name: ns,
				Cluster: clientcmdapi.Cluster{
					Server:                   serverAddr,
					CertificateAuthorityData: ca,
				},
			},
		},
		AuthInfos: []clientcmdapi.NamedAuthInfo{
			{
				Name: saName,
				AuthInfo: clientcmdapi.AuthInfo{
					Token: token,
				},
			},
		},
		Contexts: []clientcmdapi.NamedContext{
			{
				Name: ns,
				Context: clientcmdapi.Context{
					Cluster:   ns,
					AuthInfo:  saName,
					Namespace: ns,
				},
			},
		},
		CurrentContext: ns,
	}
	jsonBytes, err := json.Marshal(kubeStruct)
	must(err)
	kubeYaml, err := yaml2.JSONToYAML(jsonBytes)
	must(err)
	fmt.Println(string(kubeYaml))
}
