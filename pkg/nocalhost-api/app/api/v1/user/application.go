package user

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

func ListByApplication(c *gin.Context) {
	// userId, _ := c.Get("userId")
	users, err := listByApplication(c, true)
	if err != nil {
		api.SendResponse(c, err, nil)
	}

	api.SendResponse(c, nil, users)
}

func ListNotInApplication(c *gin.Context) {
	// userId, _ := c.Get("userId")

	users, err := listByApplication(c, false)
	if err != nil {
		api.SendResponse(c, nil, users)
	}

	api.SendResponse(c, nil, users)
}

func listByApplication(c *gin.Context, inApp bool) ([]*model.UserList, error) {
	applicationId := cast.ToUint64(c.Param("id"))
	applicationUsers, err := service.Svc.ApplicationUser().ListByApplicationId(c, applicationId)

	if err != nil {
		log.Error(err)
		return nil, errno.ErrListApplicationUser
	}

	userList, err := service.Svc.UserSvc().GetUserList(c)
	if err != nil {
		log.Error(err)
		return nil, errno.ErrListApplicationUser
	}

	set := map[uint64]interface{}{}
	for _, au := range applicationUsers {
		set[au.UserId] = "-"
	}

	result := []*model.UserList{}
	for _, user := range userList {
		_, ok := set[user.ID]

		if inApp && ok {
			result = append(result, user)
		} else if !inApp && !ok {
			result = append(result, user)
		}
	}

	return result, nil
}
