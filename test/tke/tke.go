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

package tke

import (
	"fmt"
	"github.com/google/uuid"
	"io/ioutil"
	"nocalhost/pkg/nhctl/log"
	"os"
	"strings"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
)

func CreateK8s() *task {
	id := os.Getenv("TKE_SECRET_ID")
	key := os.Getenv("TKE_SECRET_KEY")
	if id == "" || key == "" {
		panic("SECRET_ID or SECRET_KEY is null, please make sure you have set it correctly")
	}
	t := NewTask(id, key)
	t.CreateTKE()
	t.WaitClusterToBeReady()
	t.WaitInstanceToBeReady()
	retryTimes := 250
	var ok = false
	for i := 0; i < retryTimes; i++ {
		t.EnableInternetAccess()
		if t.WaitNetworkToBeReady() {
			ok = true
			break
		}
		time.Sleep(time.Second * 2)
	}
	if !ok {
		panic("Enable internet access error, please checkout you tke cluster")
	}
	t.GetKubeconfig()
	return t
}
func DeleteTke(t *task) {
	t.Delete()
}

func NewTask(secretId, secretKey string) *task {
	return &task{
		secretId:  secretId,
		secretKey: secretKey,
	}
}

type task struct {
	secretKey string
	secretId  string
	clusterId string
	client    *tke.Client
}

var DefaultConfig = defaultConfig{
	vpcId:                     "vpc-6z0motnx",
	subNet:                    "subnet-g7vr4qce",
	k8sVersion:                "1.18.4",
	os:                        "centos7.6.0_x64",
	clusterType:               "MANAGED_CLUSTER",
	zone:                      "ap-guangzhou-3",
	instanceType:              "SA2.SMALL4",
	nodeRole:                  "WORKER",
	internetMaxBandwidthOut:   100,
	maxNum:                    32,
	ignoreClusterCIDRConflict: true,
	endpoint:                  "tke.tencentcloudapi.com",
	region:                    "ap-guangzhou",
	cidrPattern:               "10.%d.0.0/24",
}

type defaultConfig struct {
	vpcId                     string
	subNet                    string
	k8sVersion                string
	os                        string
	clusterType               string
	zone                      string
	instanceType              string
	nodeRole                  string
	internetMaxBandwidthOut   int
	maxNum                    uint64
	ignoreClusterCIDRConflict bool
	endpoint                  string
	region                    string
	cidrPattern               string
}

func (t *task) GetClient() *tke.Client {
	if t.client == nil {
		credential := common.NewCredential(t.secretId, t.secretKey)
		cpf := profile.NewClientProfile()
		cpf.HttpProfile.Endpoint = DefaultConfig.endpoint
		client, _ := tke.NewClient(credential, DefaultConfig.region, cpf)
		t.client = client
	}
	return t.client
}

func (t *task) CreateTKE() {
	retryTimes := 250
	clusterName := "test-" + uuid.New().String()

	request := tke.NewCreateClusterRequest()
	request.ClusterType = &DefaultConfig.clusterType
	request.ClusterBasicSettings = &tke.ClusterBasicSettings{
		ClusterOs:      &DefaultConfig.os,
		ClusterVersion: &DefaultConfig.k8sVersion,
		ClusterName:    &clusterName,
		VpcId:          &DefaultConfig.vpcId,
	}
	request.ClusterCIDRSettings = &tke.ClusterCIDRSettings{
		//ClusterCIDR:               &cidr,
		MaxClusterServiceNum:      &DefaultConfig.maxNum,
		MaxNodePodNum:             &DefaultConfig.maxNum,
		IgnoreClusterCIDRConflict: &DefaultConfig.ignoreClusterCIDRConflict,
	}
	configStr := `
{
   "VirtualPrivateCloud":{
      "SubnetId":"%s",
      "VpcId":"%s"
   },
   "Placement":{
      "Zone":"%s"
   },
   "InstanceType":"%s",
   "SystemDisk":{
      "DiskType":"CLOUD_PREMIUM"
   },
   "DataDisks":[
      {
         "DiskType":"CLOUD_PREMIUM",
         "DiskSize":50
      }
   ],
   "InstanceCount":1,
   "InternetAccessible":{
      "PublicIpAssigned":true,
      "InternetMaxBandwidthOut":%d
   }
}
`
	configStr = fmt.Sprintf(configStr,
		DefaultConfig.subNet,
		DefaultConfig.vpcId,
		DefaultConfig.zone,
		DefaultConfig.instanceType,
		DefaultConfig.internetMaxBandwidthOut)
	request.RunInstancesForNode = []*tke.RunInstancesForNode{{
		NodeRole:         &DefaultConfig.nodeRole,
		RunInstancesPara: []*string{&configStr},
	}}

	for i := 0; i < retryTimes; i++ {
		time.Sleep(time.Second * 5)
		cidr := fmt.Sprintf(DefaultConfig.cidrPattern, i)
		request.ClusterCIDRSettings.ClusterCIDR = &cidr

		response, err := t.GetClient().CreateCluster(request)
		if err != nil {
			var s string
			if strings.Contains(err.Error(), "CIDR_CONFLICT_WITH") {
				s = "cidr conflicted, retrying " + string(rune(i))
			} else {
				s = fmt.Sprintf("error has returned: %s, retrying", err.Error())
			}
			log.Info(s)
			continue
		}
		if response != nil && response.Response != nil && response.Response.ClusterId != nil {
			t.clusterId = *response.Response.ClusterId
			log.Info("create tke successfully, clusterId: " + t.clusterId)
			return
		} else {
			log.Info("response is null, retrying " + string(rune(i)))
		}
	}
}

func (t *task) WaitClusterToBeReady() {
	request := tke.NewDescribeClustersRequest()
	request.ClusterIds = []*string{&t.clusterId}
	for {
		time.Sleep(time.Second * 5)
		response, err := t.GetClient().DescribeClusters(request)
		if err != nil {
			log.Infof("wait Cluster: %s to be ready occurs a error, info: %v", t.clusterId, err)
			continue
		}
		if response != nil &&
			response.Response != nil &&
			response.Response.Clusters != nil &&
			len(response.Response.Clusters) != 0 &&
			"Running" == *response.Response.Clusters[0].ClusterStatus {
			log.Infof("cluster: %s ready", t.clusterId)
			return
		} else {
			log.Infof("cluster: %s not ready", t.clusterId)
			continue
		}
	}
}

func (t task) WaitInstanceToBeReady() {
	request := tke.NewDescribeClusterInstancesRequest()
	request.ClusterId = &t.clusterId
	for {
		time.Sleep(time.Second * 5)
		response, err := t.GetClient().DescribeClusterInstances(request)
		if err != nil {
			log.Infof("wait cluster: %s instance to be ready occurs error, info: %v", t.clusterId, err.Error())
			continue
		}
		if response != nil &&
			response.Response != nil &&
			response.Response.InstanceSet != nil &&
			len(response.Response.InstanceSet) != 0 &&
			response.Response.InstanceSet[0] != nil &&
			response.Response.InstanceSet[0].InstanceState != nil &&
			*response.Response.InstanceSet[0].InstanceState == "running" {
			log.Infof("cluster: %s instance ready", t.clusterId)
			return
		}
		log.Infof("cluster: %s instance not ready", t.clusterId)
	}
}

func (t *task) EnableInternetAccess() {
	request := tke.NewCreateClusterEndpointVipRequest()
	request.ClusterId = &t.clusterId
	policy := "0.0.0.0/0"
	request.SecurityPolicies = []*string{&policy}

	for {
		time.Sleep(time.Second * 5)
		if _, err := t.GetClient().CreateClusterEndpointVip(request); err != nil {
			log.Infof("error has returned: %v", err)
			continue
		}
		log.Infof("enabled cluster: %s internet access", t.clusterId)
		return
	}
}

func (t *task) WaitNetworkToBeReady() bool {
	request := tke.NewDescribeClusterEndpointVipStatusRequest()
	request.ClusterId = &t.clusterId
	for {
		time.Sleep(time.Second * 5)
		response, err := t.GetClient().DescribeClusterEndpointVipStatus(request)
		if err != nil {
			log.Infof("Wait cluster: %s network to be ready error: %v", t.clusterId, err.Error())
			continue
		}
		if response == nil || response.Response == nil || response.Response.Status == nil {
			log.Infof("waiting for cluster: %s network ready", t.clusterId)
			continue
		}
		switch *response.Response.Status {
		case "Created":
			log.Infof("cluster: %s, network endpoint create successfully", t.clusterId)
			return true
		case "CreateFailed":
			log.Infof("cluster: %s network endpoint create failed, retrying, response: %s",
				t.clusterId, response.ToJsonString())
			return false
		case "Creating":
			log.Infof("cluster: %s, network endpoint creating", t.clusterId)
			continue
		default:
			log.Infof("cluster: %s, network endpoint status: %s, waiting to be ready",
				t.clusterId, *response.Response.Status)
			return false
		}
	}
}

func (t *task) GetKubeconfig() {
	request := tke.NewDescribeClusterKubeconfigRequest()
	request.ClusterId = &t.clusterId
	for {
		time.Sleep(time.Second * 5)
		response, err := t.GetClient().DescribeClusterKubeconfig(request)
		if err != nil || response == nil || response.Response == nil || response.Response.Kubeconfig == nil {
			log.Info("Retry to get kubeconfig")
			continue
		}
		var fi *os.File
		if fi, err = ioutil.TempFile("", "*kubeconfig"); err != nil {
			log.Infof("create temp kubeconfig file error: %v", err)
			continue
		}
		if _, err = fi.WriteString(*response.Response.Kubeconfig); err != nil {
			log.Infof("write kubeconfig to temp file error: %v", err)
			continue
		}
		if err = fi.Sync(); err != nil {
			log.Infof("flush kubeconfig to disk error: %v", err)
			continue
		}
		_ = os.Setenv("KUBECONFIG_PATH", fi.Name())
		log.Info(fi.Name())
		return
	}
}

func (t *task) Delete() {
	mode := "terminate"
	request := tke.NewDeleteClusterRequest()
	request.ClusterId = &t.clusterId
	request.InstanceDeleteMode = &mode
	for {
		time.Sleep(time.Second * 5)
		_, err := t.GetClient().DeleteCluster(request)
		if err != nil {
			log.Infof("delete tke cluster: %s error, retrying, error info: %v", t.clusterId, err)
			continue
		}
		log.Infof("delete tke cluster: %s successfully", t.clusterId)
		return
	}
}
