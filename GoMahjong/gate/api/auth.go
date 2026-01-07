package api

import (
	"common/http"
	"common/rpc"
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	userpb "user/pb"
)

func LoginHandler(c *http.Context) error {
	var req struct {
		Account  string `json:"account" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.BadRequest("请求参数错误")
		return nil
	}

	ctx := context.Background()
	resp, err := rpc.UserClient.Login(ctx, &userpb.LoginParams{
		Account:       req.Account,
		Password:      req.Password,
		LoginPlatform: 0,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Unauthenticated:
				c.Unauthorized("用户名或密码错误")
			case codes.NotFound:
				c.NotFound("用户不存在")
			default:
				c.InternalServerError("登录失败，稍后重试")
			}
			return nil
		}
		c.InternalServerError("登录失败，稍后重试")
		return nil
	}

	token := generateToken(resp.Uid)
	c.Success(map[string]any{"token": token})

	return nil
}

func RegisterHandler(c *http.Context) error {
	var req struct {
		Account  string `json:"account" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.BadRequest("请求参数错误")
		return nil
	}

	ctx := context.Background()
	resp, err := rpc.UserClient.Register(ctx, &userpb.RegisterParams{
		Account:       req.Account,
		Password:      req.Password,
		LoginPlatform: 1,
		SmsCode:       "0",
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.AlreadyExists:
				c.ErrorWithCode(40001, "用户名已存在")
			case codes.InvalidArgument:
				c.BadRequest(st.Message())
			default:
				c.InternalServerError("注册失败，请稍后重试")
			}
			return nil
		}
		c.InternalServerError("注册失败，请稍后重试")
		return nil
	}

	token := generateToken(resp.Uid)
	c.Success(map[string]any{"token": token})
	return nil
}

func RefreshTokenHandler(c *http.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.BadRequest("请求参数错误")
		return nil
	}

	token := "uid-token"
	c.Success(map[string]any{"token": token})
	return nil
}

// fixme 生成 token
func generateToken(uid string) string {
	return "uid-token"
}
