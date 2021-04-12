package service_account

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
	"sync"
)

type SaAuthorizeRequest struct {
	ClusterId *uint64 `json:"cluster_id" binding:"required"`
	UserId    *uint64 `json:"user_id" binding:"required"`
	SpaceName string  `json:"space_name" binding:"required"`
}

func Authorize(c *gin.Context) {
	var req SaAuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("bind service account authorizeRequest params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	service.AuthorizeNsToUser(c, *req.ClusterId, *req.UserId, req.SpaceName)
}

func ListAuthorization(c *gin.Context) {
	userId, err := ginbase.LoginUser(c)
	if err != nil {
		api.SendResponse(c, errno.ErrLoginRequired, nil)
		return
	}

	// optimization required
	clusters, err := service.Svc.ClusterSvc().GetList(c)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}

	user, err := service.Svc.UserSvc().GetUserByID(c, userId)
	if err != nil {
		api.SendResponse(c, errno.ErrUserNotFound, nil)
		return
	}

	devSpaces, err := service.Svc.ClusterUser().GetList(context.TODO(), model.ClusterUserModel{})
	if err != nil {
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}

	spacenameMap := map[uint64]map[string]string{}
	for _, space := range devSpaces {
		m, ok := spacenameMap[space.ClusterId]
		if !ok {
			m = map[string]string{}
		}

		m[space.Namespace] = space.SpaceName
		spacenameMap[space.ClusterId] = m
	}

	var result []*ServiceAccountModel
	var lock sync.Mutex
	wg := sync.WaitGroup{}
	wg.Add(len(clusters))

	for _, cluster := range clusters {
		cluster := cluster
		go func() {

			defer wg.Done()
			// new client go
			clientGo, err := clientgo.NewAdminGoClient([]byte(cluster.KubeConfig))
			if err != nil {
				return
			}

			secret, err := clientGo.GetSecret(user.SaName, "default")
			if err != nil {
				return
			}

			kubeConfig, _, _ := setupcluster.NewDevKubeConfigReader(secret, cluster.Server, "default").GetCA().GetToken().AssembleDevKubeConfig().ToYamlString()

			crb, err := clientGo.GetClusterRoleBindingByLabel(fmt.Sprintf("%s=%s", service.NOCALHOST_SA_KEY, user.SaName))
			if err != nil {
				return
			}

			var nss []NS
			for _, item := range crb.Items {
				var spaceName = fmt.Sprintf("Nocalhost-%s", item.Namespace)

				m, ok := spacenameMap[cluster.ID]
				if ok {
					s, ok := m[item.Namespace]
					if ok {
						spaceName = s
					}
				}

				nss = append(nss, NS{
					SpaceName: spaceName,
					Namespace: item.Namespace,
				})
			}

			lock.Lock()
			result = append(result, &ServiceAccountModel{
				KubeConfig:   kubeConfig,
				StorageClass: cluster.StorageClass,
				NS:           nss,
			})
			lock.Unlock()
		}()
	}

	wg.Wait()
	api.SendResponse(c, nil, result)
}


type ServiceAccountModel struct {
	KubeConfig   string `json:"kubeconfig"`
	StorageClass string `json:"storage_class"`
	NS           []NS   `json:"ns"`
}

type NS struct {
	Namespace string `json:"namespace"`
	SpaceName string `json:"spacename"`
}
