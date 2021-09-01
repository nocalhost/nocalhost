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
	"io/ioutil"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"path/filepath"
	"strings"
)

func init() {
	rootCmd.AddCommand(kubeconfigGenerateCmd)
}

var kubeconfigGenerateCmd = &cobra.Command{
	Use:   "kubeconfig-generate",
	Short: "Generate a kubeconfig for specify namespace",
	Long:  `Generate a kubeconfig for specify namespace`,
	Run: func(cmd *cobra.Command, args []string) {
		if nameSpace == "" {
			log.Fatal("--namespace must be specify")
		}

		if kubeConfig == "" {
			kubeConfig = filepath.Join(utils.GetHomePath(), ".kube", "config")
		}
		GenKubeconfig(kubeConfig, nameSpace)
	},
}

func GenKubeconfig(kube, ns string) {
	configBytes, err := ioutil.ReadFile(kube)
	must(err)

	clientGo, err := clientgo.NewAdminGoClient(configBytes)
	must(err)

	k8sClient, err := clientgoutils.NewClientGoUtils(kube, ns)
	must(err)

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
	//must(service.CreateOrUpdateRoleBindingINE(clientGo, ns, saName, ns, rb, role))

	restConfig, err := clientcmd.BuildConfigFromFlags("", kube)
	must(err)
	serverAddr := restConfig.Host

	sa, err := clientGo.GetServiceAccount(saName, ns)
	if err != nil || len(sa.Secrets) == 0 {
		return
	}

	secret, err := clientGo.GetSecret(sa.Secrets[0].Name, ns)
	must(err)

	ca := secret.Data[global.NocalhostDevServiceAccountSecretCaKey]
	if len(ca) == 0 {
		log.Fatal("Failed to get ca")
	}

	token := string(secret.Data[global.NocalhostDevServiceAccountTokenKey])
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
