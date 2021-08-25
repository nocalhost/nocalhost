/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package setupcluster

import (
	"encoding/json"
	"errors"
	yaml2 "github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

type DevKubeConfigReader interface {
	GetToken() *devKubeConfigReader
	GetCA() *devKubeConfigReader
	AssembleDevKubeConfig() *devKubeConfigReader
	ToYamlString() (string, error, error)
	ToStruct() (*clientcmdapi.Config, error, error)
}

type devKubeConfigReader struct {
	inServer         string
	inNameSpace      string
	secret           *corev1.Secret
	token            string
	ca               []byte
	kubeConfigYaml   string
	kubeConfigStruct *clientcmdapi.Config
	err              error
	errCode          error
}

func (d *devKubeConfigReader) GetToken() *devKubeConfigReader {
	d.token = string(d.secret.Data[global.NocalhostDevServiceAccountTokenKey])
	if d.token == "" {
		d.err = errors.New("get dev serviceAccount token err")
		d.errCode = errno.ErrBindSecretTokenGetErr
	}
	return d
}

func (d *devKubeConfigReader) GetCA() *devKubeConfigReader {
	d.ca = d.secret.Data[global.NocalhostDevServiceAccountSecretCaKey]
	if len(d.ca) == 0 {
		d.err = errors.New("get dev serviceAccount ca err")
		d.errCode = errno.ErrBindSecretCAGetErr
	}
	return d
}

func (d *devKubeConfigReader) AssembleDevKubeConfig() *devKubeConfigReader {
	devUserName := global.NocalhostDevServiceAccountName
	d.kubeConfigStruct = &clientcmdapi.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []clientcmdapi.NamedCluster{
			{
				Name: d.inNameSpace,
				Cluster: clientcmdapi.Cluster{
					Server:                   d.inServer,
					CertificateAuthorityData: d.ca,
				},
			},
		},
		AuthInfos: []clientcmdapi.NamedAuthInfo{
			{
				Name: devUserName,
				AuthInfo: clientcmdapi.AuthInfo{
					Token: d.token,
				},
			},
		},
		Contexts: []clientcmdapi.NamedContext{
			{
				Name: d.inNameSpace,
				Context: clientcmdapi.Context{
					Cluster:   d.inNameSpace,
					AuthInfo:  devUserName,
					Namespace: d.inNameSpace,
				},
			},
		},
		CurrentContext: d.inNameSpace,
	}
	return d
}

func (d *devKubeConfigReader) ToYamlString() (string, error, error) {
	jsonBytes, err := json.Marshal(&d.kubeConfigStruct)
	if err != nil {
		d.err = errors.New("cluster dev: kubeconfig struct encode to json error")
		d.errCode = errno.ErrBindServiceAccountKubeConfigJsonEncodeErr
	}
	kubeYaml, err := yaml2.JSONToYAML(jsonBytes)
	if err != nil {
		d.err = errors.New("cluster dev: json to yaml fail")
		d.errCode = errno.ErrBindServiceAccountStructEncodeErr
	}
	return string(kubeYaml), d.err, d.errCode
}

func (d *devKubeConfigReader) ToStruct() (*clientcmdapi.Config, error, error) {
	return d.kubeConfigStruct, d.err, d.errCode
}

func NewDevKubeConfigReader(s *corev1.Secret, server, namespace string) DevKubeConfigReader {
	return &devKubeConfigReader{
		secret:      s,
		inServer:    server,
		inNameSpace: namespace,
	}
}
