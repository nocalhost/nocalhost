/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"io/ioutil"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/util"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
)

// Create TKE Cluster
// TKE package is use for manage TKE Cluster when test has been start
// Each Github PR will create TKE Cluster for running test case
func (t *task) Create() (string, error) {
	t.deleteIdlingCluster()
	t.createTKE()
	t.waitClusterToBeReady()
	t.waitInstanceToBeReady()
	t.enableInternetAccess()
	t.waitNetworkToBeReady()
	return t.getKubeconfig(), nil
}

// DeleteTke Delete TKE Cluster
func DeleteTke(t *task) {
	t.Delete()
}

// newTask create a task with secret id and secret key
func newTask(secretId, secretKey string) *task {
	return &task{
		secretId:  secretId,
		secretKey: secretKey,
	}
}

func NewTKE() Cluster {
	id := os.Getenv(util.SecretId)
	key := os.Getenv(util.SecretKey)
	if id == "" || key == "" {
		panic(errors.New("SECRET_ID or SECRET_KEY is null, please make sure you have set it correctly"))
	}
	return newTask(id, key)
}

type task struct {
	secretKey string
	secretId  string
	clusterId string
	client    *tke.Client
}

// guangzhou area
var _ = defaultConfig{
	vpcId:                     "vpc-6z0motnx",
	subNet:                    "subnet-g7vr4qce",
	k8sVersion:                "1.18.4",
	os:                        "centos7.6.0_x64",
	clusterType:               "MANAGED_CLUSTER",
	zone:                      "ap-guangzhou-3",
	instanceType:              "SA2.SMALL4",
	diskType:                  "CLOUD_PREMIUM",
	nodeRole:                  "WORKER",
	internetMaxBandwidthOut:   100,
	maxNum:                    32,
	ignoreClusterCIDRConflict: true,
	endpoint:                  "tke.tencentcloudapi.com",
	region:                    "ap-guangzhou",
	cidrPattern:               "10.%d.0.0/24",
}

// siliconValley
var _ = defaultConfig{
	vpcId:                     "vpc-ejqejan1",
	subNet:                    "subnet-nei8cjdw",
	k8sVersion:                "1.18.4",
	os:                        "centos7.6.0_x64",
	clusterType:               "MANAGED_CLUSTER",
	zone:                      "na-siliconvalley-1",
	instanceType:              "C3.LARGE8",
	diskType:                  "CLOUD_SSD",
	nodeRole:                  "WORKER",
	internetMaxBandwidthOut:   100,
	maxNum:                    32,
	ignoreClusterCIDRConflict: true,
	endpoint:                  "tke.tencentcloudapi.com",
	region:                    "na-siliconvalley",
	cidrPattern:               "10.%d.0.0/16",
}

var DefaultConfig = defaultConfig{
	vpcId:                     "vpc-93iqnk7q",
	subNet:                    "subnet-d7m18ag1",
	k8sVersion:                "1.18.4",
	os:                        "centos7.6.0_x64",
	clusterType:               "MANAGED_CLUSTER",
	zone:                      "ap-hongkong-2",
	instanceType:              "S2.MEDIUM8",
	diskType:                  "CLOUD_PREMIUM",
	nodeRole:                  "WORKER",
	internetMaxBandwidthOut:   100,
	maxNum:                    256,
	ignoreClusterCIDRConflict: true,
	endpoint:                  "tke.tencentcloudapi.com",
	region:                    "ap-hongkong",
	cidrPattern:               "10.%d.0.0/16",
}

type defaultConfig struct {
	vpcId                     string
	subNet                    string
	k8sVersion                string
	os                        string
	clusterType               string
	zone                      string
	instanceType              string
	diskType                  string
	nodeRole                  string
	internetMaxBandwidthOut   int
	maxNum                    uint64
	ignoreClusterCIDRConflict bool
	endpoint                  string
	region                    string
	cidrPattern               string
}

// getClient of openapi
func (t *task) getClient() *tke.Client {
	if t.client == nil {
		credential := common.NewCredential(t.secretId, t.secretKey)
		cpf := profile.NewClientProfile()
		cpf.HttpProfile.Endpoint = DefaultConfig.endpoint
		client, _ := tke.NewClient(credential, DefaultConfig.region, cpf)
		t.client = client
	}
	return t.client
}

// createTKE Create TKE Cluster
func (t *task) createTKE() {

	retryTimes := 250
	clusterName := "test-" + uuid.New().String() + "(" + runtime.GOOS + ")"
	os.Setenv("TKE_NAME", clusterName)

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

	// RunInstancesPara use for CVM type
	p := Parameter{
		VirtualPrivateCloud: VirtualPrivateCloud{
			SubnetID: DefaultConfig.subNet,
			VpcID:    DefaultConfig.vpcId,
		},
		Placement: Placement{
			Zone: DefaultConfig.zone,
		},
		InstanceType: DefaultConfig.instanceType,
		SystemDisk: SystemDisk{
			DiskType: DefaultConfig.diskType,
		},
		DataDisks: []DataDisks{{
			DiskType: DefaultConfig.diskType,
			DiskSize: 50,
		}},
		InstanceCount: 1,
		InternetAccessible: InternetAccessible{
			PublicIPAssigned:        true,
			InternetMaxBandwidthOut: DefaultConfig.internetMaxBandwidthOut,
		},
		ActionTimer: ActionTimer{
			Externals:   Externals{ReleaseAddress: true},
			TimerAction: "TerminateInstances",
			ActionTime:  time.Now().Add(time.Hour * 8).Add(time.Minute * 60).Format("2006-01-02 15:04:05"),
		},
	}
	bytes, _ := json.Marshal(p)
	configStr := string(bytes)
	request.RunInstancesForNode = []*tke.RunInstancesForNode{{
		NodeRole:         &DefaultConfig.nodeRole,
		RunInstancesPara: []*string{&configStr},
	}}

	for i := 0; i < retryTimes; i++ {
		time.Sleep(time.Second * 5)
		cidr := fmt.Sprintf(DefaultConfig.cidrPattern, i)
		request.ClusterCIDRSettings.ClusterCIDR = &cidr

		response, err := t.getClient().CreateCluster(request)
		if err != nil {
			var s string
			if strings.Contains(err.Error(), "CIDR_CONFLICT_WITH") {
				s = "cidr conflicted, retrying " + strconv.Itoa(i)
			} else if strings.Contains(err.Error(), "ResourceInsufficient.SpecifiedInstanceType") {
				goto instanceTypeRetry
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

instanceTypeRetry:
	for _, instanceType := range []string{"SA2.MEDIUM8", "SA2.MEDIUM4", "SA2.LARGE8", "S2.MEDIUM8", "S2.LARGE8",
		"S2.MEDIUM4", "S5.MEDIUM8", "S5.LARGE8", "S5.MEDIUM4", "SA2.SMALL4", "SA2.SMALL2"} {
		for {
			p.InstanceType = instanceType
			bytes, _ = json.Marshal(p)
			configStr = string(bytes)
			request.RunInstancesForNode = []*tke.RunInstancesForNode{{
				NodeRole:         &DefaultConfig.nodeRole,
				RunInstancesPara: []*string{&configStr},
			}}
			response, err := t.getClient().CreateCluster(request)
			if err != nil {
				if strings.Contains(err.Error(), "ResourceInsufficient.SpecifiedInstanceType") {
					log.Infof("The specified type: %s of instance is understocked", instanceType)
					break
				}
				continue
			}
			if response != nil && response.Response != nil && response.Response.ClusterId != nil {
				t.clusterId = *response.Response.ClusterId
				log.Infof("create tke successfully, clusterId: %s, instanceType: %s", t.clusterId, instanceType)
				return
			}
		}
	}
}

// waitClusterToBeReady include TKE create success
func (t *task) waitClusterToBeReady() {
	request := tke.NewDescribeClustersRequest()
	request.ClusterIds = []*string{&t.clusterId}
	for {
		time.Sleep(time.Second * 5)
		response, err := t.getClient().DescribeClusters(request)
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

// deleteIdlingCluster Delete idling tke cluster
func (t *task) deleteIdlingCluster() {
	request := tke.NewDescribeClustersRequest()
	request.ClusterIds = []*string{}
	response, err := t.getClient().DescribeClusters(request)
	if err != nil {
		log.Infof("error while delete useless cluster, response: %v, err: %v", response, err)
		return
	}
	if response != nil &&
		response.Response != nil &&
		response.Response.Clusters != nil &&
		len(response.Response.Clusters) != 0 {
		var wg sync.WaitGroup
		for _, cluster := range response.Response.Clusters {
			if "Idling" == *cluster.ClusterStatus &&
				strings.Contains(*cluster.ClusterName, "test") &&
				*cluster.ClusterNodeNum == 0 {
				wg.Add(1)
				go func(clusterId string) {
					defer wg.Done()
					t.deleteOne(clusterId)
				}(*cluster.ClusterId)
			}
		}
		wg.Wait()
	}
}

// waitInstanceToBeReady wait instance to be ready
func (t task) waitInstanceToBeReady() {
	request := tke.NewDescribeClusterInstancesRequest()
	request.ClusterId = &t.clusterId
	for {
		time.Sleep(time.Second * 5)
		response, err := t.getClient().DescribeClusterInstances(request)
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

// enableInternetAccess open ip white list
func (t *task) enableInternetAccess() {
	request := tke.NewCreateClusterEndpointVipRequest()
	request.ClusterId = &t.clusterId
	policy := "0.0.0.0/0"
	request.SecurityPolicies = []*string{&policy}

	for {
		time.Sleep(time.Second * 5)
		if _, err := t.getClient().CreateClusterEndpointVip(request); err != nil {
			log.Infof("error has returned: %v", err)
			continue
		}
		log.Infof("enabled cluster: %s internet access", t.clusterId)
		return
	}
}

// waitNetworkToBeReady wait connection ready
func (t *task) waitNetworkToBeReady() bool {
	request := tke.NewDescribeClusterEndpointVipStatusRequest()
	request.ClusterId = &t.clusterId
	for {
		time.Sleep(time.Second * 5)
		response, err := t.getClient().DescribeClusterEndpointVipStatus(request)
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
			continue
		}
	}
}

// getKubeconfig
func (t *task) getKubeconfig() string {
	request := tke.NewDescribeClusterKubeconfigRequest()
	request.ClusterId = &t.clusterId
	for {
		time.Sleep(time.Second * 5)
		response, err := t.getClient().DescribeClusterKubeconfig(request)
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
		_ = os.Setenv(util.KubeconfigPath, fi.Name())
		return fi.Name()
	}
}

// Delete Cluster
func (t *task) Delete() {
	t.deleteOne(t.clusterId)
}
func (t *task) deleteOne(clusterId string) {
	if len(clusterId) == 0 {
		clusterId = t.clusterId
	}
	mode := "terminate"
	cbs := "CBS"
	request := tke.NewDeleteClusterRequest()
	request.ClusterId = &clusterId
	request.InstanceDeleteMode = &mode
	option := tke.ResourceDeleteOption{
		ResourceType: &cbs,
		DeleteMode:   &mode,
	}
	request.ResourceDeleteOptions = []*tke.ResourceDeleteOption{&option}
	for {
		time.Sleep(time.Second * 1)
		_, err := t.getClient().DeleteCluster(request)
		if err != nil {
			log.Infof("delete tke cluster: %s error, retrying, error info: %v", clusterId, err)
			continue
		}
		log.Infof("delete tke cluster: %s successfully", clusterId)
		return
	}
}

// Parameter struct
type Parameter struct {
	VirtualPrivateCloud VirtualPrivateCloud `json:"VirtualPrivateCloud"`
	Placement           Placement           `json:"Placement"`
	InstanceType        string              `json:"InstanceType"`
	SystemDisk          SystemDisk          `json:"SystemDisk"`
	DataDisks           []DataDisks         `json:"DataDisks"`
	InstanceCount       int                 `json:"InstanceCount"`
	InternetAccessible  InternetAccessible  `json:"InternetAccessible"`
	ActionTimer         ActionTimer         `json:"ActionTimer"`
}

type ActionTimer struct {
	Externals   Externals `json:"Externals"`
	TimerAction string    `json:"TimerAction"`
	ActionTime  string    `json:"ActionTime"`
}

type Externals struct {
	ReleaseAddress bool `json:"ReleaseAddress"`
}

// VirtualPrivateCloud struct
type VirtualPrivateCloud struct {
	SubnetID string `json:"SubnetId"`
	VpcID    string `json:"VpcId"`
}

// Placement struct
type Placement struct {
	Zone string `json:"Zone"`
}

// SystemDisk struct
type SystemDisk struct {
	DiskType string `json:"DiskType"`
}

// DataDisks struct
type DataDisks struct {
	DiskType string `json:"DiskType"`
	DiskSize int    `json:"DiskSize"`
}

// InternetAccessible struct
type InternetAccessible struct {
	PublicIPAssigned        bool `json:"PublicIpAssigned"`
	InternetMaxBandwidthOut int  `json:"InternetMaxBandwidthOut"`
}
