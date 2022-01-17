/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/imroc/req"
	"github.com/pkg/errors"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/request"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/util"
	"os"
	"runtime"
	"strconv"
	"time"
)

type vCluster struct {
	host     string
	email    string
	password string
	id       uint64
}

func NewDefaultVCluster() Cluster {
	addr := os.Getenv(util.NocalhostVClusterHostForTest)
	email := os.Getenv(util.NocalhostVClusterEmailForTest)
	passwd := os.Getenv(util.NocalhostVClusterPasswordForTest)
	if len(email) == 0 {
		email = app.DefaultInitAdminUserName
	}
	if len(passwd) == 0 {
		passwd = app.DefaultInitPassword
	}
	return NewVCluster(addr, email, passwd)
}

func NewVCluster(addr, email, password string) Cluster {
	return &vCluster{
		host:     addr,
		email:    email,
		password: password,
	}
}

// Create  vCluster called test, and get this vCluster
func (bc *vCluster) Create() (string, error) {
	res := request.NewReq("", "", "", "", 0)
	res.SpecifyService(bc.host)
	res.Login(bc.email, bc.password)
	header := req.Header{"Accept": "application/json", "Authorization": "Bearer " + res.AuthToken, "content-type": "text/plain"}
	// create vCluster
	spaceName := runtime.GOOS + "_testcase_" + uuid.NewString()
	resp, err := req.New().Post(
		res.BaseUrl+util.WebDevSpace, header,
		fmt.Sprintf(`{"cluster_id":1,"cluster_admin":0,"user_id":1,"space_name":"%s","space_resource_limit":null,
"dev_space_type":3,"virtual_cluster":{"service_type":"NodePort","version":"","values":null}}`, spaceName),
	)
	if err != nil || resp == nil {
		log.Infof("Get kubeconfig error, err: %v", err)
		return "", err
	}
	log.Infof(resp.String())
	var c CreateDevSpaceResponse
	err = resp.ToJSON(&c)
	bc.id = c.Data.ID

	// wait for vCluster ready
	for {
		resp, err := req.New().Get(
			res.BaseUrl+util.WebDevSpaceStatus+"?ids="+strconv.Itoa(int(bc.id)), header,
		)
		if err == nil {
			var result DevSpaceStatusResponse
			resp.ToJSON(&result)
			if status, ok := result.Data[bc.id]; ok && status.VirtualCluster.Status == "Ready" {
				break
			}
		}
		time.Sleep(time.Second * 1)
	}

	r, err := req.New().Get(res.BaseUrl+fmt.Sprintf(util.WebDevSpaceDetail, bc.id), header)
	if err != nil {
		log.Infof("get kubeconfig error, err: %v, response: %v, retrying", err, r)
		return "", err
	}
	re := DevSpaceDetailResponse{}
	err = r.ToJSON(&re)
	if re.Code != 0 || re.Data.KubeConfig == "" {
		toString, _ := r.ToString()
		log.Infof("get kubeconfig response error, response: %v, string: %s, retrying", re, toString)
		return "", nil
	}
	config := re.Data.KubeConfig
	if config == "" {
		return "", errors.New("Can't not get kubeconfig from webserver, please check your code")
	}
	f, _ := ioutil.TempFile("", "*newkubeconfig")
	_, _ = f.WriteString(config)
	_ = f.Sync()
	return f.Name(), nil
}

func (bc *vCluster) Delete() {
	res := request.NewReq("", "", "", "", 0)
	res.SpecifyService(bc.host)
	//res.Login(app.DefaultInitAdminUserName, app.DefaultInitPassword)
	res.Login(bc.email, bc.password)
	header := req.Header{"Accept": "application/json", "Authorization": "Bearer " + res.AuthToken, "content-type": "text/plain"}
	// create vCluster
	resp, err := req.New().Delete(res.BaseUrl+util.WebDevSpace+"/"+strconv.Itoa(int(bc.id)), header)
	if err != nil || resp == nil {
		log.Infof("Get kubeconfig error, err: %v", err)
	}
}

type DevSpaceStatus struct {
	VirtualCluster VirtualClusterInfo `json:"virtual_cluster"`
}

type VirtualClusterInfo struct {
	Status      string             `json:"status"`
	Events      string             `json:"events,omitempty"`
	ServiceType corev1.ServiceType `json:"service_type"`
	Version     string             `json:"version"`
	Values      string             `json:"values"`
}

type DevSpaceStatusResponse struct {
	Code    int                        `json:"code"`
	Message string                     `json:"message"`
	Data    map[uint64]*DevSpaceStatus `json:"data"`
}

type CreateDevSpaceResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    model.ClusterUserModel `json:"data"`
}

type DevSpaceDetailResponse struct {
	Code    int                                       `json:"code"`
	Message string                                    `json:"message"`
	Data    model.ClusterUserJoinClusterAndAppAndUser `json:"data"`
}
