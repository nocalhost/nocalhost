/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster

import "k8s.io/client-go/tools/clientcmd"

type Nothing struct {
}

func NewNothing() Cluster {
	return &Nothing{}
}

func (n Nothing) Create() (string, error) {
	return clientcmd.RecommendedHomeFile, nil
}

func (n Nothing) Delete() {
}
