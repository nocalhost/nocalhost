package applications

import (
	"github.com/gin-gonic/gin"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Create 获取应用
// @Summary 获取应用
// @Description 用户获取应用
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":[{"id":1,"context":"application info","status":"1"}]}"
// @Router /v1/application [get]
func Get(c *gin.Context) {
	userId, _ := c.Get("userId")
	result, err := service.Svc.ApplicationSvc().GetList(c, userId.(uint64))
	if err != nil {
		log.Warnf("get Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationGet, nil)
		return
	}

	api.SendResponse(c, errno.OK, result)
}
