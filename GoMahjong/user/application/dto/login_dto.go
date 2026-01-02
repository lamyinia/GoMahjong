package dto

// LoginCommand 登录命令
type LoginCommand struct {
	Account  string
	Password string
	Platform int32
}

// LoginResponse 登录响应
type LoginResponse struct {
	UID string
}
