package applications

// 创建应用请求体
type CreateAppRequest struct {
	Context string `json:"context" binding:"required"`
	Status  *uint8 `json:"status" binding:"required"`
}
