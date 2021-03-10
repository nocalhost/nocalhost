package application_user

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

func ListByApplication(c *gin.Context) {
	// userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	result, err := service.Svc.ApplicationUser().ListByApplicationId(c, applicationId)

	if err != nil {
		api.SendResponse(c, errno.ErrListApplicationUser, nil)
		return
	}

	api.SendResponse(c, nil, result)
}

func ListByUser(c *gin.Context) {
	// userId, _ := c.Get("userId")
	userId := cast.ToUint64(c.Param("id"))
	result, err := service.Svc.ApplicationUser().ListByUserId(c, userId)

	if err != nil {
		api.SendResponse(c, errno.ErrListApplicationUser, nil)
		return
	}

	api.SendResponse(c, nil, result)
}


//todo
func ListUnAuthUsers(c *gin.Context) {

}
