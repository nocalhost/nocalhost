package errno

//nolint: golint
var (
	// Common errors
	OK                  = &Errno{Code: 0, Message: "OK"}
	InternalServerError = &Errno{Code: 10001, Message: "Internal server error"}
	ErrBind             = &Errno{Code: 10002, Message: "Error occurred while binding the request body to the struct."}
	ErrParam            = &Errno{Code: 10003, Message: "参数有误"}
	ErrSignParam        = &Errno{Code: 10004, Message: "签名参数有误"}

	// user errors
	ErrUserNotFound          = &Errno{Code: 20102, Message: "The user was not found."}
	ErrTokenInvalid          = &Errno{Code: 20103, Message: "token 无效或登陆过期，请重新登陆"}
	ErrEmailOrPassword       = &Errno{Code: 20111, Message: "邮箱或密码错误"}
	ErrTwicePasswordNotMatch = &Errno{Code: 20112, Message: "两次密码输入不一致"}
	ErrRegisterFailed        = &Errno{Code: 20113, Message: "注册失败"}

	// cluster errors
	ErrClusterCreate = &Errno{Code: 30100, Message: "添加集群失败，请重试"}

	// application errors
	ErrApplicationCreate      = &Errno{Code: 40100, Message: "添加应用失败，请重试"}
	ErrApplicationGet         = &Errno{Code: 40101, Message: "获取应用失败，请重试"}
	ErrApplicationDelete      = &Errno{Code: 40102, Message: "删除应用失败，请重试"}
	ErrApplicationUpdate      = &Errno{Code: 40103, Message: "更新应用失败，请重试"}
	ErrBindApplicationClsuter = &Errno{Code: 40104, Message: "绑定集群失败，请重试"}
)
