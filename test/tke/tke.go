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
}

func (t *task) GetClient() *tke.Client {
	if t.client == nil {
		credential := common.NewCredential(t.secretId, t.secretKey)
		cpf := profile.NewClientProfile()
		cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
		client, _ := tke.NewClient(credential, "ap-guangzhou", cpf)
		t.client = client
	}
	return t.client
}

func (t *task) CreateTKE() {
	retryTimes := 250
	cidrPattern := "10.%d.0.0/24"
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
		cidr := fmt.Sprintf(cidrPattern, i)
		request.ClusterCIDRSettings.ClusterCIDR = &cidr

		response, err := t.GetClient().CreateCluster(request)
		if err != nil {
			if strings.Contains(err.Error(), "CIDR_CONFLICT_WITH") {
				fmt.Println("cidr conflicted, retrying " + string(rune(i)))
			} else {
				fmt.Printf("error has returned: %s, retrying\n", err.Error())
			}
			continue
		}
		if response != nil && response.Response != nil && response.Response.ClusterId != nil {
			t.clusterId = *response.Response.ClusterId
			fmt.Println("create tke successfully, clusterId: " + t.clusterId)
			return
		} else {
			fmt.Println("response is null, retrying " + string(rune(i)))
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
			fmt.Printf("wait Cluster: %s to be ready occurs a error, info: %v\n", t.clusterId, err)
			continue
		}
		if response != nil &&
			response.Response != nil &&
			response.Response.Clusters != nil &&
			len(response.Response.Clusters) != 0 &&
			"Running" == *response.Response.Clusters[0].ClusterStatus {
			fmt.Printf("cluster: %s ready\n", t.clusterId)
			return
		} else {
			fmt.Printf("cluster: %s not ready\n", t.clusterId)
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
			fmt.Printf("wait cluster: %s instance to be ready occurs error, info: %v", t.clusterId, err.Error())
			continue
		}
		if response != nil &&
			response.Response != nil &&
			response.Response.InstanceSet != nil &&
			len(response.Response.InstanceSet) != 0 &&
			response.Response.InstanceSet[0] != nil &&
			response.Response.InstanceSet[0].InstanceState != nil &&
			*response.Response.InstanceSet[0].InstanceState == "running" {
			fmt.Printf("cluster: %s instance ready\n", t.clusterId)
			return
		}
		fmt.Printf("cluster: %s instance not ready\n", t.clusterId)
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
			fmt.Printf("error has returned: %v\n", err)
			continue
		} else {
			fmt.Printf("enabled cluster: %s internet access\n", t.clusterId)
			break
		}
	}
}

func (t *task) WaitNetworkToBeReady() bool {
	request := tke.NewDescribeClusterEndpointVipStatusRequest()
	request.ClusterId = &t.clusterId
	for {
		time.Sleep(time.Second * 5)
		response, err := t.GetClient().DescribeClusterEndpointVipStatus(request)
		if err != nil {
			fmt.Printf("Wait cluster: %s network to be ready error: %v\n", t.clusterId, err.Error())
			continue
		}
		if response != nil && response.Response != nil && response.Response.Status != nil {
			switch *response.Response.Status {
			case "Created":
				return true
			case "CreateFailed":
				fmt.Printf("cluster: %s network endpoint create failed, retrying, response: %s\n",
					t.clusterId, response.ToJsonString())
				return false
			}
		} else {
			fmt.Printf("Waiting for cluster: %s network ready\n", t.clusterId)
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
			fmt.Println("Retry to get kubeconfig")
		} else {
			var fi *os.File
			if fi, err = ioutil.TempFile("/tmp", "*.yaml"); err != nil {
				continue
			}
			if _, err = fi.WriteString(*response.Response.Kubeconfig); err != nil {
				continue
			}
			if err = fi.Sync(); err != nil {
				continue
			}
			_ = os.Setenv("KUBECONFIG_PATH", fi.Name())
			fmt.Println(fi.Name())
			return
		}
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
			fmt.Printf("Delete tke cluster: %s error, retrying, error info: %v\n", t.clusterId, err)
		} else {
			fmt.Printf("Delete tke cluster: %s successfully\n", t.clusterId)
			return
		}
	}
}
