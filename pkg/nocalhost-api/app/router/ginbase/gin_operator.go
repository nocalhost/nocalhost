package ginbase

import "github.com/gin-gonic/gin"

const (
	NotExist = 0
)

func IsAdmin(c *gin.Context) bool {
	isAdmin, _ := c.Get("isAdmin")
	return isAdmin.(uint64) == 1
}

func LoginUser(c *gin.Context) uint64 {
	userId, exists := c.Get("userId")
	if exists {
		return userId.(uint64)
	} else {
		return NotExist
	}
}
