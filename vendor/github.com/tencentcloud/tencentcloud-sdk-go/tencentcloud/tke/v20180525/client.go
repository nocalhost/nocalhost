// Copyright (c) 2017-2018 THL A29 Limited, a Tencent company. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v20180525

import (
    "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
    tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
    "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

const APIVersion = "2018-05-25"

type Client struct {
    common.Client
}

// Deprecated
func NewClientWithSecretId(secretId, secretKey, region string) (client *Client, err error) {
    cpf := profile.NewClientProfile()
    client = &Client{}
    client.Init(region).WithSecretId(secretId, secretKey).WithProfile(cpf)
    return
}

func NewClient(credential *common.Credential, region string, clientProfile *profile.ClientProfile) (client *Client, err error) {
    client = &Client{}
    client.Init(region).
        WithCredential(credential).
        WithProfile(clientProfile)
    return
}


func NewAcquireClusterAdminRoleRequest() (request *AcquireClusterAdminRoleRequest) {
    request = &AcquireClusterAdminRoleRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "AcquireClusterAdminRole")
    return
}

func NewAcquireClusterAdminRoleResponse() (response *AcquireClusterAdminRoleResponse) {
    response = &AcquireClusterAdminRoleResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 通过此接口，可以获取集群的tke:admin的ClusterRole，即管理员角色，可以用于CAM侧高权限的用户，通过CAM策略给予子账户此接口权限，进而可以通过此接口直接获取到kubernetes集群内的管理员角色。
func (c *Client) AcquireClusterAdminRole(request *AcquireClusterAdminRoleRequest) (response *AcquireClusterAdminRoleResponse, err error) {
    if request == nil {
        request = NewAcquireClusterAdminRoleRequest()
    }
    response = NewAcquireClusterAdminRoleResponse()
    err = c.Send(request, response)
    return
}

func NewAddExistedInstancesRequest() (request *AddExistedInstancesRequest) {
    request = &AddExistedInstancesRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "AddExistedInstances")
    return
}

func NewAddExistedInstancesResponse() (response *AddExistedInstancesResponse) {
    response = &AddExistedInstancesResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 添加已经存在的实例到集群
func (c *Client) AddExistedInstances(request *AddExistedInstancesRequest) (response *AddExistedInstancesResponse, err error) {
    if request == nil {
        request = NewAddExistedInstancesRequest()
    }
    response = NewAddExistedInstancesResponse()
    err = c.Send(request, response)
    return
}

func NewAddNodeToNodePoolRequest() (request *AddNodeToNodePoolRequest) {
    request = &AddNodeToNodePoolRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "AddNodeToNodePool")
    return
}

func NewAddNodeToNodePoolResponse() (response *AddNodeToNodePoolResponse) {
    response = &AddNodeToNodePoolResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 将集群内节点移入节点池
func (c *Client) AddNodeToNodePool(request *AddNodeToNodePoolRequest) (response *AddNodeToNodePoolResponse, err error) {
    if request == nil {
        request = NewAddNodeToNodePoolRequest()
    }
    response = NewAddNodeToNodePoolResponse()
    err = c.Send(request, response)
    return
}

func NewCheckInstancesUpgradeAbleRequest() (request *CheckInstancesUpgradeAbleRequest) {
    request = &CheckInstancesUpgradeAbleRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "CheckInstancesUpgradeAble")
    return
}

func NewCheckInstancesUpgradeAbleResponse() (response *CheckInstancesUpgradeAbleResponse) {
    response = &CheckInstancesUpgradeAbleResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 检查给定节点列表中哪些是可升级的 
func (c *Client) CheckInstancesUpgradeAble(request *CheckInstancesUpgradeAbleRequest) (response *CheckInstancesUpgradeAbleResponse, err error) {
    if request == nil {
        request = NewCheckInstancesUpgradeAbleRequest()
    }
    response = NewCheckInstancesUpgradeAbleResponse()
    err = c.Send(request, response)
    return
}

func NewCreateClusterRequest() (request *CreateClusterRequest) {
    request = &CreateClusterRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "CreateCluster")
    return
}

func NewCreateClusterResponse() (response *CreateClusterResponse) {
    response = &CreateClusterResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 创建集群
func (c *Client) CreateCluster(request *CreateClusterRequest) (response *CreateClusterResponse, err error) {
    if request == nil {
        request = NewCreateClusterRequest()
    }
    response = NewCreateClusterResponse()
    err = c.Send(request, response)
    return
}

func NewCreateClusterAsGroupRequest() (request *CreateClusterAsGroupRequest) {
    request = &CreateClusterAsGroupRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "CreateClusterAsGroup")
    return
}

func NewCreateClusterAsGroupResponse() (response *CreateClusterAsGroupResponse) {
    response = &CreateClusterAsGroupResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 为已经存在的集群创建伸缩组
func (c *Client) CreateClusterAsGroup(request *CreateClusterAsGroupRequest) (response *CreateClusterAsGroupResponse, err error) {
    if request == nil {
        request = NewCreateClusterAsGroupRequest()
    }
    response = NewCreateClusterAsGroupResponse()
    err = c.Send(request, response)
    return
}

func NewCreateClusterEndpointRequest() (request *CreateClusterEndpointRequest) {
    request = &CreateClusterEndpointRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "CreateClusterEndpoint")
    return
}

func NewCreateClusterEndpointResponse() (response *CreateClusterEndpointResponse) {
    response = &CreateClusterEndpointResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 创建集群访问端口(独立集群开启内网/外网访问，托管集群支持开启内网访问)
func (c *Client) CreateClusterEndpoint(request *CreateClusterEndpointRequest) (response *CreateClusterEndpointResponse, err error) {
    if request == nil {
        request = NewCreateClusterEndpointRequest()
    }
    response = NewCreateClusterEndpointResponse()
    err = c.Send(request, response)
    return
}

func NewCreateClusterEndpointVipRequest() (request *CreateClusterEndpointVipRequest) {
    request = &CreateClusterEndpointVipRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "CreateClusterEndpointVip")
    return
}

func NewCreateClusterEndpointVipResponse() (response *CreateClusterEndpointVipResponse) {
    response = &CreateClusterEndpointVipResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 创建托管集群外网访问端口（老的方式，仅支持托管集群外网端口）
func (c *Client) CreateClusterEndpointVip(request *CreateClusterEndpointVipRequest) (response *CreateClusterEndpointVipResponse, err error) {
    if request == nil {
        request = NewCreateClusterEndpointVipRequest()
    }
    response = NewCreateClusterEndpointVipResponse()
    err = c.Send(request, response)
    return
}

func NewCreateClusterInstancesRequest() (request *CreateClusterInstancesRequest) {
    request = &CreateClusterInstancesRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "CreateClusterInstances")
    return
}

func NewCreateClusterInstancesResponse() (response *CreateClusterInstancesResponse) {
    response = &CreateClusterInstancesResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 扩展(新建)集群节点
func (c *Client) CreateClusterInstances(request *CreateClusterInstancesRequest) (response *CreateClusterInstancesResponse, err error) {
    if request == nil {
        request = NewCreateClusterInstancesRequest()
    }
    response = NewCreateClusterInstancesResponse()
    err = c.Send(request, response)
    return
}

func NewCreateClusterNodePoolRequest() (request *CreateClusterNodePoolRequest) {
    request = &CreateClusterNodePoolRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "CreateClusterNodePool")
    return
}

func NewCreateClusterNodePoolResponse() (response *CreateClusterNodePoolResponse) {
    response = &CreateClusterNodePoolResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 创建节点池
func (c *Client) CreateClusterNodePool(request *CreateClusterNodePoolRequest) (response *CreateClusterNodePoolResponse, err error) {
    if request == nil {
        request = NewCreateClusterNodePoolRequest()
    }
    response = NewCreateClusterNodePoolResponse()
    err = c.Send(request, response)
    return
}

func NewCreateClusterNodePoolFromExistingAsgRequest() (request *CreateClusterNodePoolFromExistingAsgRequest) {
    request = &CreateClusterNodePoolFromExistingAsgRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "CreateClusterNodePoolFromExistingAsg")
    return
}

func NewCreateClusterNodePoolFromExistingAsgResponse() (response *CreateClusterNodePoolFromExistingAsgResponse) {
    response = &CreateClusterNodePoolFromExistingAsgResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 从伸缩组创建节点池
func (c *Client) CreateClusterNodePoolFromExistingAsg(request *CreateClusterNodePoolFromExistingAsgRequest) (response *CreateClusterNodePoolFromExistingAsgResponse, err error) {
    if request == nil {
        request = NewCreateClusterNodePoolFromExistingAsgRequest()
    }
    response = NewCreateClusterNodePoolFromExistingAsgResponse()
    err = c.Send(request, response)
    return
}

func NewCreateClusterRouteRequest() (request *CreateClusterRouteRequest) {
    request = &CreateClusterRouteRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "CreateClusterRoute")
    return
}

func NewCreateClusterRouteResponse() (response *CreateClusterRouteResponse) {
    response = &CreateClusterRouteResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 创建集群路由
func (c *Client) CreateClusterRoute(request *CreateClusterRouteRequest) (response *CreateClusterRouteResponse, err error) {
    if request == nil {
        request = NewCreateClusterRouteRequest()
    }
    response = NewCreateClusterRouteResponse()
    err = c.Send(request, response)
    return
}

func NewCreateClusterRouteTableRequest() (request *CreateClusterRouteTableRequest) {
    request = &CreateClusterRouteTableRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "CreateClusterRouteTable")
    return
}

func NewCreateClusterRouteTableResponse() (response *CreateClusterRouteTableResponse) {
    response = &CreateClusterRouteTableResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 创建集群路由表
func (c *Client) CreateClusterRouteTable(request *CreateClusterRouteTableRequest) (response *CreateClusterRouteTableResponse, err error) {
    if request == nil {
        request = NewCreateClusterRouteTableRequest()
    }
    response = NewCreateClusterRouteTableResponse()
    err = c.Send(request, response)
    return
}

func NewCreateEKSClusterRequest() (request *CreateEKSClusterRequest) {
    request = &CreateEKSClusterRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "CreateEKSCluster")
    return
}

func NewCreateEKSClusterResponse() (response *CreateEKSClusterResponse) {
    response = &CreateEKSClusterResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 创建弹性集群
func (c *Client) CreateEKSCluster(request *CreateEKSClusterRequest) (response *CreateEKSClusterResponse, err error) {
    if request == nil {
        request = NewCreateEKSClusterRequest()
    }
    response = NewCreateEKSClusterResponse()
    err = c.Send(request, response)
    return
}

func NewCreatePrometheusDashboardRequest() (request *CreatePrometheusDashboardRequest) {
    request = &CreatePrometheusDashboardRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "CreatePrometheusDashboard")
    return
}

func NewCreatePrometheusDashboardResponse() (response *CreatePrometheusDashboardResponse) {
    response = &CreatePrometheusDashboardResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 创建grafana监控面板
func (c *Client) CreatePrometheusDashboard(request *CreatePrometheusDashboardRequest) (response *CreatePrometheusDashboardResponse, err error) {
    if request == nil {
        request = NewCreatePrometheusDashboardRequest()
    }
    response = NewCreatePrometheusDashboardResponse()
    err = c.Send(request, response)
    return
}

func NewCreatePrometheusTemplateRequest() (request *CreatePrometheusTemplateRequest) {
    request = &CreatePrometheusTemplateRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "CreatePrometheusTemplate")
    return
}

func NewCreatePrometheusTemplateResponse() (response *CreatePrometheusTemplateResponse) {
    response = &CreatePrometheusTemplateResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 创建一个云原生Prometheus模板实例
func (c *Client) CreatePrometheusTemplate(request *CreatePrometheusTemplateRequest) (response *CreatePrometheusTemplateResponse, err error) {
    if request == nil {
        request = NewCreatePrometheusTemplateRequest()
    }
    response = NewCreatePrometheusTemplateResponse()
    err = c.Send(request, response)
    return
}

func NewDeleteClusterRequest() (request *DeleteClusterRequest) {
    request = &DeleteClusterRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DeleteCluster")
    return
}

func NewDeleteClusterResponse() (response *DeleteClusterResponse) {
    response = &DeleteClusterResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 删除集群(YUNAPI V3版本)
func (c *Client) DeleteCluster(request *DeleteClusterRequest) (response *DeleteClusterResponse, err error) {
    if request == nil {
        request = NewDeleteClusterRequest()
    }
    response = NewDeleteClusterResponse()
    err = c.Send(request, response)
    return
}

func NewDeleteClusterAsGroupsRequest() (request *DeleteClusterAsGroupsRequest) {
    request = &DeleteClusterAsGroupsRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DeleteClusterAsGroups")
    return
}

func NewDeleteClusterAsGroupsResponse() (response *DeleteClusterAsGroupsResponse) {
    response = &DeleteClusterAsGroupsResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 删除集群伸缩组
func (c *Client) DeleteClusterAsGroups(request *DeleteClusterAsGroupsRequest) (response *DeleteClusterAsGroupsResponse, err error) {
    if request == nil {
        request = NewDeleteClusterAsGroupsRequest()
    }
    response = NewDeleteClusterAsGroupsResponse()
    err = c.Send(request, response)
    return
}

func NewDeleteClusterEndpointRequest() (request *DeleteClusterEndpointRequest) {
    request = &DeleteClusterEndpointRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DeleteClusterEndpoint")
    return
}

func NewDeleteClusterEndpointResponse() (response *DeleteClusterEndpointResponse) {
    response = &DeleteClusterEndpointResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 删除集群访问端口(独立集群开启内网/外网访问，托管集群支持开启内网访问)
func (c *Client) DeleteClusterEndpoint(request *DeleteClusterEndpointRequest) (response *DeleteClusterEndpointResponse, err error) {
    if request == nil {
        request = NewDeleteClusterEndpointRequest()
    }
    response = NewDeleteClusterEndpointResponse()
    err = c.Send(request, response)
    return
}

func NewDeleteClusterEndpointVipRequest() (request *DeleteClusterEndpointVipRequest) {
    request = &DeleteClusterEndpointVipRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DeleteClusterEndpointVip")
    return
}

func NewDeleteClusterEndpointVipResponse() (response *DeleteClusterEndpointVipResponse) {
    response = &DeleteClusterEndpointVipResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 删除托管集群外网访问端口（老的方式，仅支持托管集群外网端口）
func (c *Client) DeleteClusterEndpointVip(request *DeleteClusterEndpointVipRequest) (response *DeleteClusterEndpointVipResponse, err error) {
    if request == nil {
        request = NewDeleteClusterEndpointVipRequest()
    }
    response = NewDeleteClusterEndpointVipResponse()
    err = c.Send(request, response)
    return
}

func NewDeleteClusterInstancesRequest() (request *DeleteClusterInstancesRequest) {
    request = &DeleteClusterInstancesRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DeleteClusterInstances")
    return
}

func NewDeleteClusterInstancesResponse() (response *DeleteClusterInstancesResponse) {
    response = &DeleteClusterInstancesResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 删除集群中的实例
func (c *Client) DeleteClusterInstances(request *DeleteClusterInstancesRequest) (response *DeleteClusterInstancesResponse, err error) {
    if request == nil {
        request = NewDeleteClusterInstancesRequest()
    }
    response = NewDeleteClusterInstancesResponse()
    err = c.Send(request, response)
    return
}

func NewDeleteClusterNodePoolRequest() (request *DeleteClusterNodePoolRequest) {
    request = &DeleteClusterNodePoolRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DeleteClusterNodePool")
    return
}

func NewDeleteClusterNodePoolResponse() (response *DeleteClusterNodePoolResponse) {
    response = &DeleteClusterNodePoolResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 删除节点池
func (c *Client) DeleteClusterNodePool(request *DeleteClusterNodePoolRequest) (response *DeleteClusterNodePoolResponse, err error) {
    if request == nil {
        request = NewDeleteClusterNodePoolRequest()
    }
    response = NewDeleteClusterNodePoolResponse()
    err = c.Send(request, response)
    return
}

func NewDeleteClusterRouteRequest() (request *DeleteClusterRouteRequest) {
    request = &DeleteClusterRouteRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DeleteClusterRoute")
    return
}

func NewDeleteClusterRouteResponse() (response *DeleteClusterRouteResponse) {
    response = &DeleteClusterRouteResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 删除集群路由
func (c *Client) DeleteClusterRoute(request *DeleteClusterRouteRequest) (response *DeleteClusterRouteResponse, err error) {
    if request == nil {
        request = NewDeleteClusterRouteRequest()
    }
    response = NewDeleteClusterRouteResponse()
    err = c.Send(request, response)
    return
}

func NewDeleteClusterRouteTableRequest() (request *DeleteClusterRouteTableRequest) {
    request = &DeleteClusterRouteTableRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DeleteClusterRouteTable")
    return
}

func NewDeleteClusterRouteTableResponse() (response *DeleteClusterRouteTableResponse) {
    response = &DeleteClusterRouteTableResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 删除集群路由表
func (c *Client) DeleteClusterRouteTable(request *DeleteClusterRouteTableRequest) (response *DeleteClusterRouteTableResponse, err error) {
    if request == nil {
        request = NewDeleteClusterRouteTableRequest()
    }
    response = NewDeleteClusterRouteTableResponse()
    err = c.Send(request, response)
    return
}

func NewDeleteEKSClusterRequest() (request *DeleteEKSClusterRequest) {
    request = &DeleteEKSClusterRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DeleteEKSCluster")
    return
}

func NewDeleteEKSClusterResponse() (response *DeleteEKSClusterResponse) {
    response = &DeleteEKSClusterResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 删除弹性集群(yunapiv3)
func (c *Client) DeleteEKSCluster(request *DeleteEKSClusterRequest) (response *DeleteEKSClusterResponse, err error) {
    if request == nil {
        request = NewDeleteEKSClusterRequest()
    }
    response = NewDeleteEKSClusterResponse()
    err = c.Send(request, response)
    return
}

func NewDeletePrometheusTemplateRequest() (request *DeletePrometheusTemplateRequest) {
    request = &DeletePrometheusTemplateRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DeletePrometheusTemplate")
    return
}

func NewDeletePrometheusTemplateResponse() (response *DeletePrometheusTemplateResponse) {
    response = &DeletePrometheusTemplateResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 删除一个云原生Prometheus配置模板
func (c *Client) DeletePrometheusTemplate(request *DeletePrometheusTemplateRequest) (response *DeletePrometheusTemplateResponse, err error) {
    if request == nil {
        request = NewDeletePrometheusTemplateRequest()
    }
    response = NewDeletePrometheusTemplateResponse()
    err = c.Send(request, response)
    return
}

func NewDeletePrometheusTemplateSyncRequest() (request *DeletePrometheusTemplateSyncRequest) {
    request = &DeletePrometheusTemplateSyncRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DeletePrometheusTemplateSync")
    return
}

func NewDeletePrometheusTemplateSyncResponse() (response *DeletePrometheusTemplateSyncResponse) {
    response = &DeletePrometheusTemplateSyncResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 取消模板同步，这将会删除目标中该模板所生产的配置
func (c *Client) DeletePrometheusTemplateSync(request *DeletePrometheusTemplateSyncRequest) (response *DeletePrometheusTemplateSyncResponse, err error) {
    if request == nil {
        request = NewDeletePrometheusTemplateSyncRequest()
    }
    response = NewDeletePrometheusTemplateSyncResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeAvailableClusterVersionRequest() (request *DescribeAvailableClusterVersionRequest) {
    request = &DescribeAvailableClusterVersionRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeAvailableClusterVersion")
    return
}

func NewDescribeAvailableClusterVersionResponse() (response *DescribeAvailableClusterVersionResponse) {
    response = &DescribeAvailableClusterVersionResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 获取集群可以升级的所有版本
func (c *Client) DescribeAvailableClusterVersion(request *DescribeAvailableClusterVersionRequest) (response *DescribeAvailableClusterVersionResponse, err error) {
    if request == nil {
        request = NewDescribeAvailableClusterVersionRequest()
    }
    response = NewDescribeAvailableClusterVersionResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeClusterAsGroupOptionRequest() (request *DescribeClusterAsGroupOptionRequest) {
    request = &DescribeClusterAsGroupOptionRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeClusterAsGroupOption")
    return
}

func NewDescribeClusterAsGroupOptionResponse() (response *DescribeClusterAsGroupOptionResponse) {
    response = &DescribeClusterAsGroupOptionResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 集群弹性伸缩配置
func (c *Client) DescribeClusterAsGroupOption(request *DescribeClusterAsGroupOptionRequest) (response *DescribeClusterAsGroupOptionResponse, err error) {
    if request == nil {
        request = NewDescribeClusterAsGroupOptionRequest()
    }
    response = NewDescribeClusterAsGroupOptionResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeClusterAsGroupsRequest() (request *DescribeClusterAsGroupsRequest) {
    request = &DescribeClusterAsGroupsRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeClusterAsGroups")
    return
}

func NewDescribeClusterAsGroupsResponse() (response *DescribeClusterAsGroupsResponse) {
    response = &DescribeClusterAsGroupsResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 集群关联的伸缩组列表
func (c *Client) DescribeClusterAsGroups(request *DescribeClusterAsGroupsRequest) (response *DescribeClusterAsGroupsResponse, err error) {
    if request == nil {
        request = NewDescribeClusterAsGroupsRequest()
    }
    response = NewDescribeClusterAsGroupsResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeClusterEndpointStatusRequest() (request *DescribeClusterEndpointStatusRequest) {
    request = &DescribeClusterEndpointStatusRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeClusterEndpointStatus")
    return
}

func NewDescribeClusterEndpointStatusResponse() (response *DescribeClusterEndpointStatusResponse) {
    response = &DescribeClusterEndpointStatusResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 查询集群访问端口状态(独立集群开启内网/外网访问，托管集群支持开启内网访问)
func (c *Client) DescribeClusterEndpointStatus(request *DescribeClusterEndpointStatusRequest) (response *DescribeClusterEndpointStatusResponse, err error) {
    if request == nil {
        request = NewDescribeClusterEndpointStatusRequest()
    }
    response = NewDescribeClusterEndpointStatusResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeClusterEndpointVipStatusRequest() (request *DescribeClusterEndpointVipStatusRequest) {
    request = &DescribeClusterEndpointVipStatusRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeClusterEndpointVipStatus")
    return
}

func NewDescribeClusterEndpointVipStatusResponse() (response *DescribeClusterEndpointVipStatusResponse) {
    response = &DescribeClusterEndpointVipStatusResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 查询集群开启端口流程状态(仅支持托管集群外网端口)
func (c *Client) DescribeClusterEndpointVipStatus(request *DescribeClusterEndpointVipStatusRequest) (response *DescribeClusterEndpointVipStatusResponse, err error) {
    if request == nil {
        request = NewDescribeClusterEndpointVipStatusRequest()
    }
    response = NewDescribeClusterEndpointVipStatusResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeClusterInstancesRequest() (request *DescribeClusterInstancesRequest) {
    request = &DescribeClusterInstancesRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeClusterInstances")
    return
}

func NewDescribeClusterInstancesResponse() (response *DescribeClusterInstancesResponse) {
    response = &DescribeClusterInstancesResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

//  查询集群下节点实例信息 
func (c *Client) DescribeClusterInstances(request *DescribeClusterInstancesRequest) (response *DescribeClusterInstancesResponse, err error) {
    if request == nil {
        request = NewDescribeClusterInstancesRequest()
    }
    response = NewDescribeClusterInstancesResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeClusterKubeconfigRequest() (request *DescribeClusterKubeconfigRequest) {
    request = &DescribeClusterKubeconfigRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeClusterKubeconfig")
    return
}

func NewDescribeClusterKubeconfigResponse() (response *DescribeClusterKubeconfigResponse) {
    response = &DescribeClusterKubeconfigResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 获取集群的kubeconfig文件，不同子账户获取自己的kubeconfig文件，该文件中有每个子账户自己的kube-apiserver的客户端证书，默认首次调此接口时候创建客户端证书，时效20年，未授予任何权限，如果是集群所有者或者主账户，则默认是cluster-admin权限。
func (c *Client) DescribeClusterKubeconfig(request *DescribeClusterKubeconfigRequest) (response *DescribeClusterKubeconfigResponse, err error) {
    if request == nil {
        request = NewDescribeClusterKubeconfigRequest()
    }
    response = NewDescribeClusterKubeconfigResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeClusterNodePoolDetailRequest() (request *DescribeClusterNodePoolDetailRequest) {
    request = &DescribeClusterNodePoolDetailRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeClusterNodePoolDetail")
    return
}

func NewDescribeClusterNodePoolDetailResponse() (response *DescribeClusterNodePoolDetailResponse) {
    response = &DescribeClusterNodePoolDetailResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 查询节点池详情
func (c *Client) DescribeClusterNodePoolDetail(request *DescribeClusterNodePoolDetailRequest) (response *DescribeClusterNodePoolDetailResponse, err error) {
    if request == nil {
        request = NewDescribeClusterNodePoolDetailRequest()
    }
    response = NewDescribeClusterNodePoolDetailResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeClusterNodePoolsRequest() (request *DescribeClusterNodePoolsRequest) {
    request = &DescribeClusterNodePoolsRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeClusterNodePools")
    return
}

func NewDescribeClusterNodePoolsResponse() (response *DescribeClusterNodePoolsResponse) {
    response = &DescribeClusterNodePoolsResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 查询节点池列表
func (c *Client) DescribeClusterNodePools(request *DescribeClusterNodePoolsRequest) (response *DescribeClusterNodePoolsResponse, err error) {
    if request == nil {
        request = NewDescribeClusterNodePoolsRequest()
    }
    response = NewDescribeClusterNodePoolsResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeClusterRouteTablesRequest() (request *DescribeClusterRouteTablesRequest) {
    request = &DescribeClusterRouteTablesRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeClusterRouteTables")
    return
}

func NewDescribeClusterRouteTablesResponse() (response *DescribeClusterRouteTablesResponse) {
    response = &DescribeClusterRouteTablesResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 查询集群路由表
func (c *Client) DescribeClusterRouteTables(request *DescribeClusterRouteTablesRequest) (response *DescribeClusterRouteTablesResponse, err error) {
    if request == nil {
        request = NewDescribeClusterRouteTablesRequest()
    }
    response = NewDescribeClusterRouteTablesResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeClusterRoutesRequest() (request *DescribeClusterRoutesRequest) {
    request = &DescribeClusterRoutesRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeClusterRoutes")
    return
}

func NewDescribeClusterRoutesResponse() (response *DescribeClusterRoutesResponse) {
    response = &DescribeClusterRoutesResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 查询集群路由
func (c *Client) DescribeClusterRoutes(request *DescribeClusterRoutesRequest) (response *DescribeClusterRoutesResponse, err error) {
    if request == nil {
        request = NewDescribeClusterRoutesRequest()
    }
    response = NewDescribeClusterRoutesResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeClusterSecurityRequest() (request *DescribeClusterSecurityRequest) {
    request = &DescribeClusterSecurityRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeClusterSecurity")
    return
}

func NewDescribeClusterSecurityResponse() (response *DescribeClusterSecurityResponse) {
    response = &DescribeClusterSecurityResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 集群的密钥信息
func (c *Client) DescribeClusterSecurity(request *DescribeClusterSecurityRequest) (response *DescribeClusterSecurityResponse, err error) {
    if request == nil {
        request = NewDescribeClusterSecurityRequest()
    }
    response = NewDescribeClusterSecurityResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeClustersRequest() (request *DescribeClustersRequest) {
    request = &DescribeClustersRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeClusters")
    return
}

func NewDescribeClustersResponse() (response *DescribeClustersResponse) {
    response = &DescribeClustersResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 查询集群列表
func (c *Client) DescribeClusters(request *DescribeClustersRequest) (response *DescribeClustersResponse, err error) {
    if request == nil {
        request = NewDescribeClustersRequest()
    }
    response = NewDescribeClustersResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeEKSClusterCredentialRequest() (request *DescribeEKSClusterCredentialRequest) {
    request = &DescribeEKSClusterCredentialRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeEKSClusterCredential")
    return
}

func NewDescribeEKSClusterCredentialResponse() (response *DescribeEKSClusterCredentialResponse) {
    response = &DescribeEKSClusterCredentialResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 获取弹性容器集群的接入认证信息
func (c *Client) DescribeEKSClusterCredential(request *DescribeEKSClusterCredentialRequest) (response *DescribeEKSClusterCredentialResponse, err error) {
    if request == nil {
        request = NewDescribeEKSClusterCredentialRequest()
    }
    response = NewDescribeEKSClusterCredentialResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeEKSClustersRequest() (request *DescribeEKSClustersRequest) {
    request = &DescribeEKSClustersRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeEKSClusters")
    return
}

func NewDescribeEKSClustersResponse() (response *DescribeEKSClustersResponse) {
    response = &DescribeEKSClustersResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 查询弹性集群列表
func (c *Client) DescribeEKSClusters(request *DescribeEKSClustersRequest) (response *DescribeEKSClustersResponse, err error) {
    if request == nil {
        request = NewDescribeEKSClustersRequest()
    }
    response = NewDescribeEKSClustersResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeExistedInstancesRequest() (request *DescribeExistedInstancesRequest) {
    request = &DescribeExistedInstancesRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeExistedInstances")
    return
}

func NewDescribeExistedInstancesResponse() (response *DescribeExistedInstancesResponse) {
    response = &DescribeExistedInstancesResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 查询已经存在的节点，判断是否可以加入集群
func (c *Client) DescribeExistedInstances(request *DescribeExistedInstancesRequest) (response *DescribeExistedInstancesResponse, err error) {
    if request == nil {
        request = NewDescribeExistedInstancesRequest()
    }
    response = NewDescribeExistedInstancesResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeImagesRequest() (request *DescribeImagesRequest) {
    request = &DescribeImagesRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeImages")
    return
}

func NewDescribeImagesResponse() (response *DescribeImagesResponse) {
    response = &DescribeImagesResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 获取镜像信息
func (c *Client) DescribeImages(request *DescribeImagesRequest) (response *DescribeImagesResponse, err error) {
    if request == nil {
        request = NewDescribeImagesRequest()
    }
    response = NewDescribeImagesResponse()
    err = c.Send(request, response)
    return
}

func NewDescribePrometheusAgentInstancesRequest() (request *DescribePrometheusAgentInstancesRequest) {
    request = &DescribePrometheusAgentInstancesRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribePrometheusAgentInstances")
    return
}

func NewDescribePrometheusAgentInstancesResponse() (response *DescribePrometheusAgentInstancesResponse) {
    response = &DescribePrometheusAgentInstancesResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 获取关联目标集群的实例列表
func (c *Client) DescribePrometheusAgentInstances(request *DescribePrometheusAgentInstancesRequest) (response *DescribePrometheusAgentInstancesResponse, err error) {
    if request == nil {
        request = NewDescribePrometheusAgentInstancesRequest()
    }
    response = NewDescribePrometheusAgentInstancesResponse()
    err = c.Send(request, response)
    return
}

func NewDescribePrometheusAgentsRequest() (request *DescribePrometheusAgentsRequest) {
    request = &DescribePrometheusAgentsRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribePrometheusAgents")
    return
}

func NewDescribePrometheusAgentsResponse() (response *DescribePrometheusAgentsResponse) {
    response = &DescribePrometheusAgentsResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 获取被关联集群列表
func (c *Client) DescribePrometheusAgents(request *DescribePrometheusAgentsRequest) (response *DescribePrometheusAgentsResponse, err error) {
    if request == nil {
        request = NewDescribePrometheusAgentsRequest()
    }
    response = NewDescribePrometheusAgentsResponse()
    err = c.Send(request, response)
    return
}

func NewDescribePrometheusAlertHistoryRequest() (request *DescribePrometheusAlertHistoryRequest) {
    request = &DescribePrometheusAlertHistoryRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribePrometheusAlertHistory")
    return
}

func NewDescribePrometheusAlertHistoryResponse() (response *DescribePrometheusAlertHistoryResponse) {
    response = &DescribePrometheusAlertHistoryResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 获取告警历史
func (c *Client) DescribePrometheusAlertHistory(request *DescribePrometheusAlertHistoryRequest) (response *DescribePrometheusAlertHistoryResponse, err error) {
    if request == nil {
        request = NewDescribePrometheusAlertHistoryRequest()
    }
    response = NewDescribePrometheusAlertHistoryResponse()
    err = c.Send(request, response)
    return
}

func NewDescribePrometheusAlertRuleRequest() (request *DescribePrometheusAlertRuleRequest) {
    request = &DescribePrometheusAlertRuleRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribePrometheusAlertRule")
    return
}

func NewDescribePrometheusAlertRuleResponse() (response *DescribePrometheusAlertRuleResponse) {
    response = &DescribePrometheusAlertRuleResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 获取告警规则列表
func (c *Client) DescribePrometheusAlertRule(request *DescribePrometheusAlertRuleRequest) (response *DescribePrometheusAlertRuleResponse, err error) {
    if request == nil {
        request = NewDescribePrometheusAlertRuleRequest()
    }
    response = NewDescribePrometheusAlertRuleResponse()
    err = c.Send(request, response)
    return
}

func NewDescribePrometheusOverviewsRequest() (request *DescribePrometheusOverviewsRequest) {
    request = &DescribePrometheusOverviewsRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribePrometheusOverviews")
    return
}

func NewDescribePrometheusOverviewsResponse() (response *DescribePrometheusOverviewsResponse) {
    response = &DescribePrometheusOverviewsResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 获取实例列表
func (c *Client) DescribePrometheusOverviews(request *DescribePrometheusOverviewsRequest) (response *DescribePrometheusOverviewsResponse, err error) {
    if request == nil {
        request = NewDescribePrometheusOverviewsRequest()
    }
    response = NewDescribePrometheusOverviewsResponse()
    err = c.Send(request, response)
    return
}

func NewDescribePrometheusTargetsRequest() (request *DescribePrometheusTargetsRequest) {
    request = &DescribePrometheusTargetsRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribePrometheusTargets")
    return
}

func NewDescribePrometheusTargetsResponse() (response *DescribePrometheusTargetsResponse) {
    response = &DescribePrometheusTargetsResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 获取targets信息
func (c *Client) DescribePrometheusTargets(request *DescribePrometheusTargetsRequest) (response *DescribePrometheusTargetsResponse, err error) {
    if request == nil {
        request = NewDescribePrometheusTargetsRequest()
    }
    response = NewDescribePrometheusTargetsResponse()
    err = c.Send(request, response)
    return
}

func NewDescribePrometheusTemplateSyncRequest() (request *DescribePrometheusTemplateSyncRequest) {
    request = &DescribePrometheusTemplateSyncRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribePrometheusTemplateSync")
    return
}

func NewDescribePrometheusTemplateSyncResponse() (response *DescribePrometheusTemplateSyncResponse) {
    response = &DescribePrometheusTemplateSyncResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 获取模板同步信息
func (c *Client) DescribePrometheusTemplateSync(request *DescribePrometheusTemplateSyncRequest) (response *DescribePrometheusTemplateSyncResponse, err error) {
    if request == nil {
        request = NewDescribePrometheusTemplateSyncRequest()
    }
    response = NewDescribePrometheusTemplateSyncResponse()
    err = c.Send(request, response)
    return
}

func NewDescribePrometheusTemplatesRequest() (request *DescribePrometheusTemplatesRequest) {
    request = &DescribePrometheusTemplatesRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribePrometheusTemplates")
    return
}

func NewDescribePrometheusTemplatesResponse() (response *DescribePrometheusTemplatesResponse) {
    response = &DescribePrometheusTemplatesResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 拉取模板列表，默认模板将总是在最前面
func (c *Client) DescribePrometheusTemplates(request *DescribePrometheusTemplatesRequest) (response *DescribePrometheusTemplatesResponse, err error) {
    if request == nil {
        request = NewDescribePrometheusTemplatesRequest()
    }
    response = NewDescribePrometheusTemplatesResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeRegionsRequest() (request *DescribeRegionsRequest) {
    request = &DescribeRegionsRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeRegions")
    return
}

func NewDescribeRegionsResponse() (response *DescribeRegionsResponse) {
    response = &DescribeRegionsResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 获取容器服务支持的所有地域
func (c *Client) DescribeRegions(request *DescribeRegionsRequest) (response *DescribeRegionsResponse, err error) {
    if request == nil {
        request = NewDescribeRegionsRequest()
    }
    response = NewDescribeRegionsResponse()
    err = c.Send(request, response)
    return
}

func NewDescribeRouteTableConflictsRequest() (request *DescribeRouteTableConflictsRequest) {
    request = &DescribeRouteTableConflictsRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "DescribeRouteTableConflicts")
    return
}

func NewDescribeRouteTableConflictsResponse() (response *DescribeRouteTableConflictsResponse) {
    response = &DescribeRouteTableConflictsResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 查询路由表冲突列表
func (c *Client) DescribeRouteTableConflicts(request *DescribeRouteTableConflictsRequest) (response *DescribeRouteTableConflictsResponse, err error) {
    if request == nil {
        request = NewDescribeRouteTableConflictsRequest()
    }
    response = NewDescribeRouteTableConflictsResponse()
    err = c.Send(request, response)
    return
}

func NewGetUpgradeInstanceProgressRequest() (request *GetUpgradeInstanceProgressRequest) {
    request = &GetUpgradeInstanceProgressRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "GetUpgradeInstanceProgress")
    return
}

func NewGetUpgradeInstanceProgressResponse() (response *GetUpgradeInstanceProgressResponse) {
    response = &GetUpgradeInstanceProgressResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 获得节点升级当前的进度 
func (c *Client) GetUpgradeInstanceProgress(request *GetUpgradeInstanceProgressRequest) (response *GetUpgradeInstanceProgressResponse, err error) {
    if request == nil {
        request = NewGetUpgradeInstanceProgressRequest()
    }
    response = NewGetUpgradeInstanceProgressResponse()
    err = c.Send(request, response)
    return
}

func NewModifyClusterAsGroupAttributeRequest() (request *ModifyClusterAsGroupAttributeRequest) {
    request = &ModifyClusterAsGroupAttributeRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "ModifyClusterAsGroupAttribute")
    return
}

func NewModifyClusterAsGroupAttributeResponse() (response *ModifyClusterAsGroupAttributeResponse) {
    response = &ModifyClusterAsGroupAttributeResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 修改集群伸缩组属性
func (c *Client) ModifyClusterAsGroupAttribute(request *ModifyClusterAsGroupAttributeRequest) (response *ModifyClusterAsGroupAttributeResponse, err error) {
    if request == nil {
        request = NewModifyClusterAsGroupAttributeRequest()
    }
    response = NewModifyClusterAsGroupAttributeResponse()
    err = c.Send(request, response)
    return
}

func NewModifyClusterAsGroupOptionAttributeRequest() (request *ModifyClusterAsGroupOptionAttributeRequest) {
    request = &ModifyClusterAsGroupOptionAttributeRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "ModifyClusterAsGroupOptionAttribute")
    return
}

func NewModifyClusterAsGroupOptionAttributeResponse() (response *ModifyClusterAsGroupOptionAttributeResponse) {
    response = &ModifyClusterAsGroupOptionAttributeResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 修改集群弹性伸缩属性
func (c *Client) ModifyClusterAsGroupOptionAttribute(request *ModifyClusterAsGroupOptionAttributeRequest) (response *ModifyClusterAsGroupOptionAttributeResponse, err error) {
    if request == nil {
        request = NewModifyClusterAsGroupOptionAttributeRequest()
    }
    response = NewModifyClusterAsGroupOptionAttributeResponse()
    err = c.Send(request, response)
    return
}

func NewModifyClusterAttributeRequest() (request *ModifyClusterAttributeRequest) {
    request = &ModifyClusterAttributeRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "ModifyClusterAttribute")
    return
}

func NewModifyClusterAttributeResponse() (response *ModifyClusterAttributeResponse) {
    response = &ModifyClusterAttributeResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 修改集群属性
func (c *Client) ModifyClusterAttribute(request *ModifyClusterAttributeRequest) (response *ModifyClusterAttributeResponse, err error) {
    if request == nil {
        request = NewModifyClusterAttributeRequest()
    }
    response = NewModifyClusterAttributeResponse()
    err = c.Send(request, response)
    return
}

func NewModifyClusterEndpointSPRequest() (request *ModifyClusterEndpointSPRequest) {
    request = &ModifyClusterEndpointSPRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "ModifyClusterEndpointSP")
    return
}

func NewModifyClusterEndpointSPResponse() (response *ModifyClusterEndpointSPResponse) {
    response = &ModifyClusterEndpointSPResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 修改托管集群外网端口的安全策略（老的方式，仅支持托管集群外网端口）
func (c *Client) ModifyClusterEndpointSP(request *ModifyClusterEndpointSPRequest) (response *ModifyClusterEndpointSPResponse, err error) {
    if request == nil {
        request = NewModifyClusterEndpointSPRequest()
    }
    response = NewModifyClusterEndpointSPResponse()
    err = c.Send(request, response)
    return
}

func NewModifyClusterNodePoolRequest() (request *ModifyClusterNodePoolRequest) {
    request = &ModifyClusterNodePoolRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "ModifyClusterNodePool")
    return
}

func NewModifyClusterNodePoolResponse() (response *ModifyClusterNodePoolResponse) {
    response = &ModifyClusterNodePoolResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 编辑节点池
func (c *Client) ModifyClusterNodePool(request *ModifyClusterNodePoolRequest) (response *ModifyClusterNodePoolResponse, err error) {
    if request == nil {
        request = NewModifyClusterNodePoolRequest()
    }
    response = NewModifyClusterNodePoolResponse()
    err = c.Send(request, response)
    return
}

func NewModifyNodePoolDesiredCapacityAboutAsgRequest() (request *ModifyNodePoolDesiredCapacityAboutAsgRequest) {
    request = &ModifyNodePoolDesiredCapacityAboutAsgRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "ModifyNodePoolDesiredCapacityAboutAsg")
    return
}

func NewModifyNodePoolDesiredCapacityAboutAsgResponse() (response *ModifyNodePoolDesiredCapacityAboutAsgResponse) {
    response = &ModifyNodePoolDesiredCapacityAboutAsgResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 修改节点池关联伸缩组的期望实例数
func (c *Client) ModifyNodePoolDesiredCapacityAboutAsg(request *ModifyNodePoolDesiredCapacityAboutAsgRequest) (response *ModifyNodePoolDesiredCapacityAboutAsgResponse, err error) {
    if request == nil {
        request = NewModifyNodePoolDesiredCapacityAboutAsgRequest()
    }
    response = NewModifyNodePoolDesiredCapacityAboutAsgResponse()
    err = c.Send(request, response)
    return
}

func NewModifyPrometheusTemplateRequest() (request *ModifyPrometheusTemplateRequest) {
    request = &ModifyPrometheusTemplateRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "ModifyPrometheusTemplate")
    return
}

func NewModifyPrometheusTemplateResponse() (response *ModifyPrometheusTemplateResponse) {
    response = &ModifyPrometheusTemplateResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 修改模板内容
func (c *Client) ModifyPrometheusTemplate(request *ModifyPrometheusTemplateRequest) (response *ModifyPrometheusTemplateResponse, err error) {
    if request == nil {
        request = NewModifyPrometheusTemplateRequest()
    }
    response = NewModifyPrometheusTemplateResponse()
    err = c.Send(request, response)
    return
}

func NewRemoveNodeFromNodePoolRequest() (request *RemoveNodeFromNodePoolRequest) {
    request = &RemoveNodeFromNodePoolRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "RemoveNodeFromNodePool")
    return
}

func NewRemoveNodeFromNodePoolResponse() (response *RemoveNodeFromNodePoolResponse) {
    response = &RemoveNodeFromNodePoolResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 移出节点池节点，但保留在集群内
func (c *Client) RemoveNodeFromNodePool(request *RemoveNodeFromNodePoolRequest) (response *RemoveNodeFromNodePoolResponse, err error) {
    if request == nil {
        request = NewRemoveNodeFromNodePoolRequest()
    }
    response = NewRemoveNodeFromNodePoolResponse()
    err = c.Send(request, response)
    return
}

func NewSetNodePoolNodeProtectionRequest() (request *SetNodePoolNodeProtectionRequest) {
    request = &SetNodePoolNodeProtectionRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "SetNodePoolNodeProtection")
    return
}

func NewSetNodePoolNodeProtectionResponse() (response *SetNodePoolNodeProtectionResponse) {
    response = &SetNodePoolNodeProtectionResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 仅能设置节点池中处于伸缩组的节点
func (c *Client) SetNodePoolNodeProtection(request *SetNodePoolNodeProtectionRequest) (response *SetNodePoolNodeProtectionResponse, err error) {
    if request == nil {
        request = NewSetNodePoolNodeProtectionRequest()
    }
    response = NewSetNodePoolNodeProtectionResponse()
    err = c.Send(request, response)
    return
}

func NewSyncPrometheusTemplateRequest() (request *SyncPrometheusTemplateRequest) {
    request = &SyncPrometheusTemplateRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "SyncPrometheusTemplate")
    return
}

func NewSyncPrometheusTemplateResponse() (response *SyncPrometheusTemplateResponse) {
    response = &SyncPrometheusTemplateResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 同步模板到实例或者集群
func (c *Client) SyncPrometheusTemplate(request *SyncPrometheusTemplateRequest) (response *SyncPrometheusTemplateResponse, err error) {
    if request == nil {
        request = NewSyncPrometheusTemplateRequest()
    }
    response = NewSyncPrometheusTemplateResponse()
    err = c.Send(request, response)
    return
}

func NewUpdateClusterVersionRequest() (request *UpdateClusterVersionRequest) {
    request = &UpdateClusterVersionRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "UpdateClusterVersion")
    return
}

func NewUpdateClusterVersionResponse() (response *UpdateClusterVersionResponse) {
    response = &UpdateClusterVersionResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 升级集群 Master 组件到指定版本
func (c *Client) UpdateClusterVersion(request *UpdateClusterVersionRequest) (response *UpdateClusterVersionResponse, err error) {
    if request == nil {
        request = NewUpdateClusterVersionRequest()
    }
    response = NewUpdateClusterVersionResponse()
    err = c.Send(request, response)
    return
}

func NewUpdateEKSClusterRequest() (request *UpdateEKSClusterRequest) {
    request = &UpdateEKSClusterRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "UpdateEKSCluster")
    return
}

func NewUpdateEKSClusterResponse() (response *UpdateEKSClusterResponse) {
    response = &UpdateEKSClusterResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 修改弹性集群名称等属性 
func (c *Client) UpdateEKSCluster(request *UpdateEKSClusterRequest) (response *UpdateEKSClusterResponse, err error) {
    if request == nil {
        request = NewUpdateEKSClusterRequest()
    }
    response = NewUpdateEKSClusterResponse()
    err = c.Send(request, response)
    return
}

func NewUpgradeClusterInstancesRequest() (request *UpgradeClusterInstancesRequest) {
    request = &UpgradeClusterInstancesRequest{
        BaseRequest: &tchttp.BaseRequest{},
    }
    request.Init().WithApiInfo("tke", APIVersion, "UpgradeClusterInstances")
    return
}

func NewUpgradeClusterInstancesResponse() (response *UpgradeClusterInstancesResponse) {
    response = &UpgradeClusterInstancesResponse{
        BaseResponse: &tchttp.BaseResponse{},
    }
    return
}

// 给集群的一批work节点进行升级 
func (c *Client) UpgradeClusterInstances(request *UpgradeClusterInstancesRequest) (response *UpgradeClusterInstancesResponse, err error) {
    if request == nil {
        request = NewUpgradeClusterInstancesRequest()
    }
    response = NewUpgradeClusterInstancesResponse()
    err = c.Send(request, response)
    return
}
