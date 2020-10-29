package applications

import (
	"github.com/gin-gonic/gin"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Create 添加应用
// @Summary 添加应用
// @Description 用户添加应用
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param CreateAppRequest body applications.CreateAppRequest true "The application info"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/application [post]
func Create(c *gin.Context) {
	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("createApplication bind params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	userId, _ := c.Get("userId")
	err := service.Svc.ApplicationSvc().Create(c, req.Context, *req.Status, userId.(uint64))
	if err != nil {
		log.Warnf("create Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationCreate, nil)
		return
	}

	api.SendResponse(c, nil, nil)
}
