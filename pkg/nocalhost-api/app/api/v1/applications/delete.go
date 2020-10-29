package applications

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Create 删除应用
// @Summary 删除应用
// @Description 用户软删除应用
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "应用 ID"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/application/{id} [delete]
func Delete(c *gin.Context) {
	userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	err := service.Svc.ApplicationSvc().Delete(c, userId.(uint64), applicationId)
	if err != nil {
		log.Warnf("get Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationDelete, nil)
		return
	}

	api.SendResponse(c, errno.OK, nil)
}
