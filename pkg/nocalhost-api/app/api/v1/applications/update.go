package applications

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Create 编辑应用
// @Summary 编辑应用
// @Description 用户编辑应用
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "应用 ID"
// @Param CreateAppRequest body applications.CreateAppRequest true "The application info"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/application/{id} [post]
func Update(c *gin.Context) {
	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("update application bind err: %s", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	model := model.ApplicationModel{
		ID:      applicationId,
		UserId:  userId.(uint64),
		Context: req.Context,
		Status:  *req.Status,
	}
	err := service.Svc.ApplicationSvc().Update(c, &model)
	if err != nil {
		log.Warnf("update Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationUpdate, nil)
		return
	}

	api.SendResponse(c, errno.OK, nil)
}
