package dto

// RegisterCommand 注册命令
type RegisterCommand struct {
	Account  string
	Password string
	Platform int32
	SMSCode  string
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	UID string
}
